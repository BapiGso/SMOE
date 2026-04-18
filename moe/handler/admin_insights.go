package handler

import (
	"github.com/labstack/echo/v5"
)

type smoeInsight struct {
	Views     []int          `json:"Views"`
	Pages     map[string]int `json:"Pages"`
	Referrers map[string]int `json:"Referrers"`
	Browsers  map[string]int `json:"Browsers"`
	OS        map[string]int `json:"OS"`
	Devices   map[string]int `json:"Devices"`
	Countries map[string]int `json:"Countries"`
}

func Insight(c *echo.Context) error {
	// Insights disabled — return empty data
	insight := &smoeInsight{
		Views:     make([]int, 12),
		Pages:     map[string]int{},
		Referrers: map[string]int{},
		Browsers:  map[string]int{},
		OS:        map[string]int{},
		Devices:   map[string]int{},
		Countries: map[string]int{},
	}
	return c.Render(200, "insights.template", insight)
}
