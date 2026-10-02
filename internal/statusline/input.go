// Package statusline renders the Claude Code status line: every account's usage,
// the model and the pet.
package statusline

import (
	"encoding/json"
	"io"
)

// Input is the JSON Claude Code passes on stdin. A field with an unexpected type is
// skipped and the rest still decoded.
type Input struct {
	SessionID string `json:"session_id"`
	Model     struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage    *float64 `json:"used_percentage"`
		ContextWindowSize float64  `json:"context_window_size"`
		CurrentUsage      struct {
			InputTokens              float64 `json:"input_tokens"`
			CacheCreationInputTokens float64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     float64 `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	Cost struct {
		TotalCostUSD *float64 `json:"total_cost_usd"`
	} `json:"cost"`
	RateLimits *RateLimits `json:"rate_limits"`
}

type RateLimits struct {
	FiveHour *RateLimit `json:"five_hour"`
	SevenDay *RateLimit `json:"seven_day"`
}

type RateLimit struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       float64  `json:"resets_at"` // Unix seconds
}

func ParseInput(r io.Reader) Input {
	var in Input
	data, _ := io.ReadAll(r)
	json.Unmarshal(data, &in)
	return in
}

// ContextPct is how full the context window is, or nil if unknown.
func (in Input) ContextPct() *float64 {
	cw := in.ContextWindow
	if cw.UsedPercentage != nil {
		return cw.UsedPercentage
	}
	if cw.ContextWindowSize == 0 {
		return nil
	}
	u := cw.CurrentUsage
	pct := (u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens) / cw.ContextWindowSize * 100
	return &pct
}
