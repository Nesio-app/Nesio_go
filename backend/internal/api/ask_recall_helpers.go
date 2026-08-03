package api

import (
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type askRecallNode struct {
	ID    uuid.UUID `db:"id"`
	Type  string    `db:"type"`
	Title string    `db:"title"`
	Body  *string   `db:"body"`
	Tags  []string  `db:"tags"`
}

func (s *Server) queryAskRecallNodesByTypes(userID uuid.UUID, query string, types []string, limit int) ([]askRecallNode, error) {
	rows := make([]askRecallNode, 0)
	if len(types) == 0 {
		return rows, nil
	}
	err := s.store.DB.Select(&rows, `
		SELECT id, type, title, body, tags
		FROM life_nodes
		WHERE user_id = $1
		  AND type = ANY($2::text[])
		  AND (
			 title ILIKE '%' || $3 || '%'
			 OR COALESCE(body, '') ILIKE '%' || $3 || '%'
			 OR EXISTS (
				 SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || $3 || '%'
			 )
		  )
		ORDER BY updated_at DESC
		LIMIT $4
	`, userID, pq.Array(types), query, limit)
	return rows, err
}

func askSnippetFromTitleBody(title string, body *string) string {
	snippet := title
	if body != nil && strings.TrimSpace(*body) != "" {
		snippet += "\n" + strings.TrimSpace(*body)
	}
	return snippet
}
