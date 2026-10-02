package statusline

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

var t0 = time.Unix(1_800_000_000, 0)

func writeCreds(t *testing.T, dir, token string, expiresAtMs float64, plan string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	creds := map[string]any{"claudeAiOauth": map[string]any{
		"accessToken": token, "refreshToken": "refresh-" + token, "expiresAt": expiresAtMs,
		"scopes": []string{"user:inference", "user:profile"}, "subscriptionType": plan,
	}, "other": "kept"}
	if err := fsutil.WriteJSON(filepath.Join(dir, ".credentials.json"), creds, false); err != nil {
		t.Fatal(err)
	}
}

const usageBody = `{"five_hour":{"utilization":31.0,"resets_at":"2027-01-15T10:05:00.123+00:00"},"seven_day":{"utilization":8,"resets_at":"2027-01-18T12:00:00Z"},"extra":1}`

func fetcher(srv *httptest.Server) *Fetcher {
	return &Fetcher{UsageURL: srv.URL + "/usage", TokenURL: srv.URL + "/token", Client: srv.Client()}
}

func TestFetchSuccess(t *testing.T) {
	dir := t.TempDir()
	writeCreds(t, dir, "tok", 9e12, "max")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.Header.Get("anthropic-beta") != "oauth-2025-04-20" {
			t.Errorf("headers: %v", r.Header)
		}
		w.Write([]byte(usageBody))
	}))
	defer srv.Close()
	e, err := fetcher(srv).Fetch(dir, false, t0)
	if err != nil {
		t.Fatal(err)
	}
	if e.Plan != "max" || *e.FiveHour.Utilization != 31 || e.SevenDay.ResetsAt != "2027-01-18T12:00:00Z" || e.FetchedAt != 1_800_000_000 {
		t.Fatalf("got %+v", e)
	}
}

func TestFetchErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	dir := t.TempDir()
	if _, err := fetcher(srv).Fetch(dir, false, t0); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("no creds: %v", err)
	}
	writeCreds(t, dir, "tok", 9e12, "pro")
	var he *HTTPError
	if _, err := fetcher(srv).Fetch(dir, false, t0); !errors.As(err, &he) || he.Code != 401 || err.Error() != "HTTP 401" {
		t.Errorf("401: %v", err)
	}
}

func TestFetchExpiredActiveTokenIsLeftToClaude(t *testing.T) {
	dir := t.TempDir()
	writeCreds(t, dir, "tok", float64(t0.Unix())*1000, "pro")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s", r.URL)
	}))
	defer srv.Close()
	if _, err := fetcher(srv).Fetch(dir, true, t0); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("got %v", err)
	}
}

func TestFetchRefreshesInactiveToken(t *testing.T) {
	dir := t.TempDir()
	writeCreds(t, dir, "old", float64(t0.Unix()+30)*1000, "pro")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["grant_type"] != "refresh_token" || body["refresh_token"] != "refresh-old" ||
				body["scope"] != "user:inference user:profile" || body["client_id"] == "" {
				t.Errorf("token request: %v", body)
			}
			w.Write([]byte(`{"access_token":"new","refresh_token":"refresh-new","expires_in":3600}`))
		case "/usage":
			if r.Header.Get("Authorization") != "Bearer new" {
				t.Errorf("usage called with %q", r.Header.Get("Authorization"))
			}
			w.Write([]byte(usageBody))
		}
	}))
	defer srv.Close()
	if _, err := fetcher(srv).Fetch(dir, false, t0); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".credentials.json")
	data, _ := os.ReadFile(path)
	text := string(data)
	if !strings.Contains(text, `"accessToken":"new"`) || !strings.Contains(text, `"refreshToken":"refresh-new"`) ||
		!strings.Contains(text, `"other":"kept"`) || !strings.Contains(text, `"expiresAt":1800003600000`) {
		t.Fatalf("credentials: %s", text)
	}
	if fi, _ := os.Stat(path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm %v", fi.Mode().Perm())
	}
}

func TestFromRateLimits(t *testing.T) {
	var in Input
	json.Unmarshal([]byte(`{"rate_limits":{"five_hour":{"used_percentage":9,"resets_at":1800007470},"seven_day":{"used_percentage":43,"resets_at":1800273570}}}`), &in)
	e := FromRateLimits(in.RateLimits, "pro", t0)
	if e == nil || *e.FiveHour.Utilization != 9 || e.FiveHour.ResetsAt != "2027-01-15T10:04:30Z" || *e.SevenDay.Utilization != 43 || e.Plan != "pro" {
		t.Fatalf("got %+v", e)
	}
	if FromRateLimits(nil, "pro", t0) != nil {
		t.Fatal("no rate limits means no entry")
	}
}

func TestParseInputAndContextPct(t *testing.T) {
	in := ParseInput(strings.NewReader(`{"session_id":"s","context_window":{"context_window_size":200000,"current_usage":{"input_tokens":1000,"cache_read_input_tokens":19000,"cache_creation_input_tokens":0,"output_tokens":5}}}`))
	if in.SessionID != "s" || in.ContextPct() == nil || *in.ContextPct() != 10 {
		t.Fatalf("got %+v", in)
	}
	if bad := ParseInput(strings.NewReader("not json")); bad.SessionID != "" || bad.ContextPct() != nil {
		t.Fatal("garbage input should parse as empty")
	}
}

func TestCacheLegacyKeysAndForget(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "usage-cache.json"), []byte(`{"A":{"plan":"pro","fetched_at":1},"B":{"plan":"max","fetched_at":2}}`), 0o644)
	c := LoadCache(dir)
	if c["a"] == nil || c["b"].Plan != "max" || c["A"] != nil {
		t.Fatalf("got %v", c)
	}
	c.Save(dir)
	Forget(dir, "b")
	if c := LoadCache(dir); c["b"] != nil || c["a"] == nil {
		t.Fatalf("after forget: %v", c)
	}
}
