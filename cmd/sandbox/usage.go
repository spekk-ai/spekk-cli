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

// usageTotals keeps the last reported totals for each Claude session. Workers
// can end between turns, so this state belongs to the client, not a worker.
type usageTotals struct {
	mu     sync.Mutex
	totals map[string]usageBaseline
}

type usageBaseline struct {
	cost   float64
	counts [4]int64
	valid  bool
}

// measure records cost even when token counts are absent or malformed. A
// valid later result can establish a new token baseline for that session.
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
	counts, countsOK := modelUsageCounts(event["modelUsage"])

	u.mu.Lock()
	if u.totals == nil {
		u.totals = make(map[string]usageBaseline)
	}
	previous, known := u.totals[sessionID]
	u.totals[sessionID] = usageBaseline{total, counts, countsOK}
	u.mu.Unlock()

	cost := total
	if resumeID != "" {
		if resumeID != sessionID || !known {
			return turnUsage{}, false
		}
		cost -= previous.cost
	}
	if cost < 0 || cost > 1000 || math.IsNaN(cost) || math.IsInf(cost, 0) {
		return turnUsage{}, false
	}
	if !countsOK || (resumeID != "" && !previous.valid) {
		return turnUsage{}, false
	}
	if resumeID != "" {
		for i := range counts {
			if counts[i] < previous.counts[i] {
				return turnUsage{}, false
			}
			counts[i] -= previous.counts[i]
		}
	}
	return turnUsage{counts[0], counts[1], counts[2], counts[3], cost}, true
}

func modelUsageCounts(value any) ([4]int64, bool) {
	models, ok := value.(map[string]any)
	if !ok || len(models) == 0 {
		return [4]int64{}, false
	}
	counts := [4]int64{}
	for _, value := range models {
		model, ok := value.(map[string]any)
		if !ok {
			return [4]int64{}, false
		}
		for i, key := range [...]string{"inputTokens", "outputTokens", "cacheReadInputTokens", "cacheCreationInputTokens"} {
			n, ok := usageCount(model[key])
			if !ok || n > maxTurnTokens-counts[i] {
				return [4]int64{}, false
			}
			counts[i] += n
		}
	}
	return counts, true
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
