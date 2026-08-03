package api

import (
	"sort"
	"sync"

	"github.com/google/uuid"
)

type scoredSearchResult struct {
	item  searchResult
	score int
}

func (s *Server) runSearchRoute(score int, sql string, userID uuid.UUID, query string, limit int) ([]scoredSearchResult, error) {
	rows := make([]searchResult, 0)
	if err := s.store.DB.Select(&rows, sql, userID, query, limit); err != nil {
		return nil, err
	}
	scored := make([]scoredSearchResult, 0, len(rows))
	for _, row := range rows {
		scored = append(scored, scoredSearchResult{item: row, score: score})
	}
	return scored, nil
}

func mergeScoredSearchResults(limit int, batches ...[]scoredSearchResult) []searchResult {
	best := map[uuid.UUID]scoredSearchResult{}
	for _, batch := range batches {
		for _, row := range batch {
			existing, ok := best[row.item.ID]
			if !ok || row.score > existing.score {
				best[row.item.ID] = row
			}
		}
	}

	ranked := make([]scoredSearchResult, 0, len(best))
	for _, row := range best {
		ranked = append(ranked, row)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].item.CreatedAt > ranked[j].item.CreatedAt
	})

	merged := make([]searchResult, 0, len(ranked))
	for _, row := range ranked {
		merged = append(merged, row.item)
	}
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}

func (s *Server) parallelSearchRecall(userID uuid.UUID, query string, limit int) ([]searchResult, error) {
	type routeResult struct {
		rows []scoredSearchResult
		err  error
	}

	routes := []struct {
		score int
		sql   string
	}{
		{
			score: 3,
			sql: `
				SELECT id, type, title, body, (attributes->>'image_url') AS image_url, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
				FROM life_nodes
				WHERE user_id = $1
				  AND (
					 title ILIKE '%' || $2 || '%'
					 OR COALESCE(body, '') ILIKE '%' || $2 || '%'
				  )
				ORDER BY updated_at DESC
				LIMIT $3
			`,
		},
		{
			score: 2,
			sql: `
				SELECT id, type, title, body, (attributes->>'image_url') AS image_url, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
				FROM life_nodes
				WHERE user_id = $1
				  AND EXISTS (
					 SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || $2 || '%'
				  )
				ORDER BY updated_at DESC
				LIMIT $3
			`,
		},
		{
			score: 1,
			sql: `
				SELECT id, type, title, body, (attributes->>'image_url') AS image_url, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
				FROM life_nodes
				WHERE user_id = $1
				  AND attributes::text ILIKE '%' || $2 || '%'
				ORDER BY updated_at DESC
				LIMIT $3
			`,
		},
		{
			score: 2,
			sql: `
				SELECT id, 'reminder' AS type, title, remind_at::text AS body, NULL::text AS image_url, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
				FROM reminders
				WHERE user_id = $1
				  AND is_done = false
				  AND (
					 title ILIKE '%' || $2 || '%'
					 OR COALESCE(source, '') ILIKE '%' || $2 || '%'
				  )
				ORDER BY remind_at DESC
				LIMIT $3
			`,
		},
		{
			score: 2,
			sql: `
				SELECT id, 'today_card' AS type, title, body, NULL::text AS image_url, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
				FROM today_cards
				WHERE user_id = $1
				  AND dismissed_at IS NULL
				  AND (
					 title ILIKE '%' || $2 || '%'
					 OR COALESCE(body, '') ILIKE '%' || $2 || '%'
				  )
				ORDER BY severity DESC, created_at DESC
				LIMIT $3
			`,
		},
	}

	resultCh := make(chan routeResult, len(routes))
	var wg sync.WaitGroup
	for _, route := range routes {
		route := route
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := s.runSearchRoute(route.score, route.sql, userID, query, limit)
			resultCh <- routeResult{rows: rows, err: err}
		}()
	}

	wg.Wait()
	close(resultCh)

	batches := make([][]scoredSearchResult, 0, len(routes))
	errorCount := 0
	var firstErr error
	for result := range resultCh {
		if result.err != nil {
			errorCount++
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}
		batches = append(batches, result.rows)
	}
	if errorCount == len(routes) {
		return nil, firstErr
	}

	return mergeScoredSearchResults(limit, batches...), nil
}
