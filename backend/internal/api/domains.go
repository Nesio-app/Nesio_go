package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type DomainOverview struct {
	Label        string `json:"label"`
	TaskCount    int    `json:"task_count"`
	MemoryCount  int    `json:"memory_count"`
	UrgentCount  int    `json:"urgent_count"`
	LatestTitles []string `json:"latest_titles"`
}

var domainLabels = []string{
	"健康", "物品", "财务", "日程", "足迹", "健身", "家务", "衣橱", "关系",
	"成长", "美味", "镜子", "资产", "目标", "剧场", "运营", "音乐", "奖励",
}

func (s *Server) handleDomainsOverview(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	result := make([]DomainOverview, 0, len(domainLabels))

	for _, label := range domainLabels {
		overview := DomainOverview{Label: label, LatestTitles: []string{}}
		_ = s.store.DB.Get(&overview.TaskCount, `
			SELECT COUNT(*) FROM life_nodes
			WHERE user_id = $1 AND type = 'task' AND domain = $2 AND status IN ('active', 'later')
		`, userID, label)
		_ = s.store.DB.Get(&overview.MemoryCount, `
			SELECT COUNT(*) FROM life_nodes
			WHERE user_id = $1 AND `+lifeNodeTypeMatchesMemory()+` AND domain = $2
		`, userID, label)
		_ = s.store.DB.Get(&overview.UrgentCount, `
			SELECT COUNT(*) FROM today_cards
			WHERE user_id = $1 AND severity >= 3 AND dismissed_at IS NULL AND title ILIKE '%' || $2 || '%'
		`, userID, label)
		var titles []string
		_ = s.store.DB.Select(&titles, `
			SELECT title FROM life_nodes
			WHERE user_id = $1 AND domain = $2
			ORDER BY updated_at DESC
			LIMIT 3
		`, userID, label)
		overview.LatestTitles = titles
		result = append(result, overview)
	}

	return c.JSON(http.StatusOK, result)
}
