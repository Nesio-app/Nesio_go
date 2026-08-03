package items

import (
	"strings"

	"github.com/Nesio-app/Nesio_go/internal/vision"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	repo *Repository
}

func NewService(db *sqlx.DB) *Service {
	return &Service{repo: NewRepository(db)}
}

func (s *Service) FindDuplicates(userID uuid.UUID, visualHash string, extraction map[string]any) ([]map[string]any, error) {
	duplicates := make([]map[string]any, 0)

	visualRows, err := s.repo.FindVisualDuplicates(userID, visualHash, 3)
	if err == nil {
		for _, row := range visualRows {
			similarity := 1.0
			if row.VisualHash != "" && visualHash != "" {
				distance := vision.HammingDistanceHex(row.VisualHash, visualHash)
				similarity = 1.0 - float64(distance)/64.0
			}
			duplicates = append(duplicates, map[string]any{
				"id":             row.ID,
				"name":           row.Name,
				"room_name":      row.RoomName,
				"container_name": row.ContainerName,
				"match_type":     "visual",
				"similarity":     similarity,
			})
		}
	}

	if len(duplicates) > 0 {
		return duplicates, nil
	}

	name, _ := extraction["name"].(string)
	if strings.TrimSpace(name) == "" {
		return duplicates, nil
	}

	nameRows, err := s.repo.FindNameDuplicates(userID, name, 3)
	if err != nil {
		return duplicates, nil
	}
	for _, row := range nameRows {
		duplicates = append(duplicates, map[string]any{
			"id":             row.ID,
			"name":           row.Name,
			"room_name":      row.RoomName,
			"container_name": row.ContainerName,
			"match_type":     "name",
		})
	}
	return duplicates, nil
}
