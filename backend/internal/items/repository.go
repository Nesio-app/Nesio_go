package items

import (
	"strings"

	"github.com/Nesio-app/Nesio_go/internal/vision"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindVisualDuplicates(userID uuid.UUID, visualHash string, limit int) ([]DuplicateRow, error) {
	rows := make([]DuplicateRow, 0)
	if strings.TrimSpace(visualHash) == "" {
		return rows, nil
	}
	visualRows := make([]DuplicateRow, 0)
	err := r.db.Select(&visualRows, `
		SELECT n.id, n.title as name, v.visual_hash, r.name AS room_name, ct.name AS container_name
		FROM item_visual_fingerprints v
		JOIN life_nodes n ON n.id = v.node_id
		LEFT JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE v.user_id = $1 AND v.visual_hash = $2
		ORDER BY v.created_at DESC
		LIMIT $3
	`, userID, strings.TrimSpace(visualHash), limit)
	if err != nil {
		return rows, err
	}
	if len(visualRows) > 0 {
		return visualRows, nil
	}

	allRows := make([]DuplicateRow, 0)
	err = r.db.Select(&allRows, `
		SELECT n.id, n.title as name, v.visual_hash, r.name AS room_name, ct.name AS container_name
		FROM item_visual_fingerprints v
		JOIN life_nodes n ON n.id = v.node_id
		LEFT JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE v.user_id = $1
		ORDER BY v.created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return rows, err
	}
	for _, candidate := range allRows {
		if vision.HammingDistanceHex(candidate.VisualHash, visualHash) <= 10 {
			rows = append(rows, candidate)
		}
		if len(rows) >= limit {
			break
		}
	}
	return rows, nil
}

func (r *Repository) FindNameDuplicates(userID uuid.UUID, name string, limit int) ([]DuplicateRow, error) {
	rows := make([]DuplicateRow, 0)
	if strings.TrimSpace(name) == "" {
		return rows, nil
	}
	err := r.db.Select(&rows, `
		SELECT n.id, n.title as name, r.name AS room_name, ct.name AS container_name
		FROM life_nodes n
		LEFT JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1 AND n.type = 'thing' AND n.title ILIKE '%' || $2 || '%'
		ORDER BY n.updated_at DESC
		LIMIT $3
	`, userID, strings.TrimSpace(name), limit)
	return rows, err
}
