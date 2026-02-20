package handler

import (
	"SMOE/moe/store"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v5"
)

type NewBangumi struct {
	Data []struct {
		UpdatedAt time.Time `json:"updated_at"`
		Comment   any       `json:"comment"`
		Tags      []any     `json:"tags"`
		Subject   struct {
			Date   string `json:"date"`
			Images struct {
				Small  string `json:"small"`
				Grid   string `json:"grid"`
				Large  string `json:"large"`
				Medium string `json:"medium"`
				Common string `json:"common"`
			} `json:"images"`
			Name         string `json:"name"`
			NameCn       string `json:"name_cn"`
			ShortSummary string `json:"short_summary"`
			Tags         []struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			} `json:"tags"`
			Score           float64 `json:"score"`
			Type            int     `json:"type"`
			ID              int     `json:"id"`
			Eps             int     `json:"eps"`
			Volumes         int     `json:"volumes"`
			CollectionTotal int     `json:"collection_total"`
			Rank            int     `json:"rank"`
		} `json:"subject"`
		SubjectID   int  `json:"subject_id"`
		VolStatus   int  `json:"vol_status"`
		EpStatus    int  `json:"ep_status"`
		SubjectType int  `json:"subject_type"`
		Type        int  `json:"type"`
		Rate        int  `json:"rate"`
		Private     bool `json:"private"`
	} `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func curlBGM(url string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Set("User-Agent", "trim21/bangumi-episode-ics (https://github.com/Trim21/bangumi-episode-calendar)")
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	m := NewBangumi{}
	if err := json.Unmarshal(body, &m); err != nil {
		return err
	}
	sort.Slice(m.Data, func(i, j int) bool {
		return m.Data[i].Subject.ID < m.Data[j].Subject.ID
	})
	bgm = bgmCache{m, time.Now().Unix()}
	return err
}

type bgmCache struct {
	NewBangumi NewBangumi
	TTL        int64
}

var bgm = bgmCache{}

// Bangumi todo https://freefrontend.com/css-cards/
func Bangumi(c *echo.Context) error {
	cfg, err := store.ReadConfig()
	if err != nil {
		return err
	}
	newAPI := "https://api.bgm.tv/v0/users/" + cfg.Bangumi.UserID + "/collections?subject_type=2&limit=100&offset=0"
	//每七天更新一下
	if time.Now().Unix()-bgm.TTL > 604800 {
		if err := curlBGM(newAPI); err != nil {
			return err
		}
	}
	return c.Render(200, "page-bangumi.template", bgm.NewBangumi)
}
