package main

import (
	"encoding/json"
	"math"
	"strconv"
	"sync"
)

const maxTurnTokens int64 = 10_000_000_000

type turnUsage struct {
	InputTokens              int64   `json:"input_tokens"`
	OutputTokens             int64   `json:"output_tokens"`
	CacheReadInputTokens     int64   `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64   `json:"cache_creation_input_tokens"`
	CostUSD                  float64 `json:"cost_usd"`
}

// usageTotals keeps the last reported cost for each Claude session. Workers
// can end between turns, so this state belongs to the client, not a worker.
type usageTotals struct {
	mu     sync.Mutex
	totals map[string]float64
}

// measure updates the baseline even if token counts are absent or malformed.
// A later resumed turn can then use its own cost without counting this one.
func (u *usageTotals) measure(sessionID, resumeID string, event map[string]any) (turnUsage, bool) {
	if sessionID == "" || event == nil {
		u.forget(resumeID)
		return turnUsage{}, false
	}
	total, ok := usageNumber(event["total_cost_usd"])
	if !ok || total < 0 {
		u.forget(resumeID)
		return turnUsage{}, false
	}
	if resumeID != "" && resumeID != sessionID {
		u.forget(resumeID)
	}

	u.mu.Lock()
	if u.totals == nil {
		u.totals = make(map[string]float64)
	}
	previous, known := u.totals[sessionID]
	u.totals[sessionID] = total
	u.mu.Unlock()

	cost := total
	if resumeID != "" {
		if resumeID != sessionID || !known {
			return turnUsage{}, false
		}
		cost -= previous
	}
	if cost < 0 || cost > 1000 || math.IsNaN(cost) || math.IsInf(cost, 0) {
		return turnUsage{}, false
	}

	usage, ok := event["usage"].(map[string]any)
	if !ok {
		return turnUsage{}, false
	}
	counts := [4]int64{}
	for i, key := range [...]string{"input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"} {
		n, ok := usageCount(usage[key])
		if !ok {
			return turnUsage{}, false
		}
		counts[i] = n
	}
	return turnUsage{counts[0], counts[1], counts[2], counts[3], cost}, true
}

func (u *usageTotals) forget(sessionID string) {
	if sessionID == "" {
		return
	}
	u.mu.Lock()
	delete(u.totals, sessionID)
	u.mu.Unlock()
}

func usageCount(value any) (int64, bool) {
	n, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	count, err := strconv.ParseInt(string(n), 10, 64)
	return count, err == nil && count >= 0 && count <= maxTurnTokens
}

func usageNumber(value any) (float64, bool) {
	n, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	cost, err := n.Float64()
	return cost, err == nil && !math.IsNaN(cost) && !math.IsInf(cost, 0)
}
