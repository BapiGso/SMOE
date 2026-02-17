package handler

import (
	"SMOE/moe/database"
	"encoding/json"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/mileusna/useragent"
)

type smoeInsight struct {
	Views     []int
	Pages     map[string]int
	Referrers map[string]int
	Browsers  map[string]int
	OS        map[string]int
	Devices   map[string]int
	Countries map[string]int
}

func (s *smoeInsight) Json() string {
	marshal, err := json.Marshal(s)
	if err != nil {
		return err.Error()
	}
	return string(marshal)
}

func Insight(c *echo.Context) error {
	qpu := &database.QPU{}
	req := &struct {
		Past int64 `query:"past"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := database.DB.Select(&qpu.Access, `SELECT * FROM smoe_insights WHERE "time" > ? ;`, time.Now().Unix()-req.Past); err != nil {
		return err
	}
	insight := &smoeInsight{
		Views:     make([]int, 12),
		Pages:     map[string]int{},
		Referrers: map[string]int{},
		Browsers:  map[string]int{},
		OS:        map[string]int{},
		Devices:   map[string]int{},
		Countries: map[string]int{},
	}
	for _, v := range qpu.Access {
		ua := useragent.Parse(v.UA)
		if !ua.Bot {
			insight.Views[v.Time%req.Past/(req.Past/12)]++
			insight.Pages[v.Path]++
			insight.Referrers[v.Referer]++
			insight.Browsers[ua.Name]++
			insight.OS[ua.OS]++
			insight.Devices[ua.Device]++
		}
	}
	return c.Render(200, "insights.template", insight.Json())
}
