package statusline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

const (
	DefaultUsageURL = "https://api.anthropic.com/api/oauth/usage"
	DefaultTokenURL = "https://platform.claude.com/v1/oauth/token"
	clientID        = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	CacheTTL        = 120 * time.Second // between fetches per account
	requestTimeout  = 4 * time.Second
)

var (
	ErrNotLoggedIn  = errors.New("not logged in")
	ErrTokenExpired = errors.New("token expired")
)

type HTTPError struct{ Code int }

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d", e.Code) }

// Window is one usage window in the api/oauth/usage format, which the cache keeps.
type Window struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    string   `json:"resets_at,omitempty"`
}

type Entry struct {
	Plan      string  `json:"plan,omitempty"`
	FiveHour  *Window `json:"five_hour"`
	SevenDay  *Window `json:"seven_day"`
	FetchedAt float64 `json:"fetched_at"`
	LastError string  `json:"last_error,omitempty"`
}

// Cache is usage-cache.json, keyed by account letter.
type Cache map[string]*Entry

func cachePath(mainDir string) string { return filepath.Join(mainDir, "usage-cache.json") }

func LoadCache(mainDir string) Cache {
	c := Cache{}
	fsutil.ReadJSON(cachePath(mainDir), &c)
	for k, v := range c {
		// The Python version keyed accounts as "A", "B".
		if low := strings.ToLower(k); low != k {
			if c[low] == nil {
				c[low] = v
			}
			delete(c, k)
		}
	}
	return c
}

func (c Cache) Save(mainDir string) error { return fsutil.WriteJSON(cachePath(mainDir), c, false) }

// Forget drops a removed account's cached usage.
func Forget(mainDir, name string) {
	c := LoadCache(mainDir)
	if _, ok := c[name]; ok {
		delete(c, name)
		c.Save(mainDir)
	}
}

func unixSecs(t time.Time) float64 { return float64(t.UnixNano()) / 1e9 }

func fromUnix(f float64) time.Time {
	sec, frac := math.Modf(f)
	return time.Unix(int64(sec), int64(frac*1e9))
}

type Fetcher struct {
	UsageURL, TokenURL string
	Client             *http.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{UsageURL: DefaultUsageURL, TokenURL: DefaultTokenURL,
		Client: &http.Client{Timeout: requestTimeout}}
}

func readCreds(dir string) (string, map[string]any, map[string]any, error) {
	path := filepath.Join(dir, ".credentials.json")
	var creds map[string]any
	if err := fsutil.ReadJSON(path, &creds); err != nil {
		return path, nil, nil, ErrNotLoggedIn
	}
	oauth, ok := creds["claudeAiOauth"].(map[string]any)
	if !ok {
		return path, nil, nil, ErrNotLoggedIn
	}
	return path, creds, oauth, nil
}

// Plan is the subscription type stored with the account's login, e.g. "pro".
func Plan(dir string) string {
	_, _, oauth, err := readCreds(dir)
	if err != nil {
		return ""
	}
	plan, _ := oauth["subscriptionType"].(string)
	return plan
}

// Fetch asks api/oauth/usage with the account's own token. An expiring token is
// refreshed only for an account that isn't running; the running Claude refreshes its own.
func (f *Fetcher) Fetch(dir string, active bool, now time.Time) (*Entry, error) {
	path, creds, oauth, err := readCreds(dir)
	if err != nil {
		return nil, err
	}
	token, _ := oauth["accessToken"].(string)
	expiresAt, _ := oauth["expiresAt"].(float64)
	if expiresAt/1000 < float64(now.Unix()+60) {
		if active {
			return nil, ErrTokenExpired
		}
		if token, err = f.refresh(path, creds, oauth, now); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(http.MethodGet, f.UsageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "claude-accounts-statusline")
	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{resp.StatusCode}
	}
	var body struct {
		FiveHour *Window `json:"five_hour"`
		SevenDay *Window `json:"seven_day"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	plan, _ := oauth["subscriptionType"].(string)
	return &Entry{Plan: plan, FiveHour: body.FiveHour, SevenDay: body.SevenDay, FetchedAt: unixSecs(now)}, nil
}

func (f *Fetcher) refresh(path string, creds, oauth map[string]any, now time.Time) (string, error) {
	var scopes []string
	if list, ok := oauth["scopes"].([]any); ok {
		for _, s := range list {
			if str, ok := s.(string); ok {
				scopes = append(scopes, str)
			}
		}
	}
	refreshToken, _ := oauth["refreshToken"].(string)
	body, _ := json.Marshal(map[string]string{"grant_type": "refresh_token",
		"refresh_token": refreshToken, "client_id": clientID, "scope": strings.Join(scopes, " ")})
	resp, err := f.Client.Post(f.TokenURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &HTTPError{resp.StatusCode}
	}
	var data struct {
		AccessToken  string  `json:"access_token"`
		RefreshToken string  `json:"refresh_token"`
		ExpiresIn    float64 `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.AccessToken == "" {
		return "", errors.New("token refresh returned no access token")
	}
	if data.ExpiresIn == 0 {
		data.ExpiresIn = 3600
	}
	oauth["accessToken"] = data.AccessToken
	if data.RefreshToken != "" {
		oauth["refreshToken"] = data.RefreshToken
	}
	oauth["expiresAt"] = float64(now.Unix()+int64(data.ExpiresIn)) * 1000
	out, err := fsutil.Marshal(creds, false)
	if err != nil {
		return "", err
	}
	return data.AccessToken, fsutil.WriteFileAtomic(path, out, 0o600)
}

// FromRateLimits turns the running account's rate_limits from the status line input
// into a cache entry, so no request is needed for it.
func FromRateLimits(rl *RateLimits, plan string, now time.Time) *Entry {
	if rl == nil || (rl.FiveHour == nil && rl.SevenDay == nil) {
		return nil
	}
	conv := func(r *RateLimit) *Window {
		if r == nil || r.UsedPercentage == nil {
			return nil
		}
		w := &Window{Utilization: r.UsedPercentage}
		if r.ResetsAt > 0 {
			w.ResetsAt = time.Unix(int64(r.ResetsAt), 0).UTC().Format(time.RFC3339)
		}
		return w
	}
	return &Entry{Plan: plan, FiveHour: conv(rl.FiveHour), SevenDay: conv(rl.SevenDay), FetchedAt: unixSecs(now)}
}
