package vision

import (
	"sort"
	"strings"
)

type DuplicateCandidate struct {
	ID         string
	Name       string
	VisualHash string
	MatchType  string
}

type ScoredDuplicate struct {
	DuplicateCandidate
	Score float64
}

// ScoreDuplicates sorts candidates by simple heuristic score.
func ScoreDuplicates(queryName, queryHash string, candidates []DuplicateCandidate) []ScoredDuplicate {
	qn := strings.ToLower(strings.TrimSpace(queryName))
	qh := strings.TrimSpace(queryHash)
	scored := make([]ScoredDuplicate, 0, len(candidates))
	for _, c := range candidates {
		score := 0.3
		if qh != "" && strings.TrimSpace(c.VisualHash) == qh {
			score += 0.5
		}
		if qn != "" {
			cn := strings.ToLower(strings.TrimSpace(c.Name))
			if cn == qn {
				score += 0.3
			} else if strings.Contains(cn, qn) || strings.Contains(qn, cn) {
				score += 0.15
			}
		}
		scored = append(scored, ScoredDuplicate{DuplicateCandidate: c, Score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	return scored
}
