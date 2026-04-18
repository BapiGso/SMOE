package handler

import (
	"SMOE/moe/store"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"sync"
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

// bangumi 缓存：并发读写安全，失败时会写入 lastAttempt 做短期退避。
type bangumiCache struct {
	mu          sync.RWMutex
	data        NewBangumi
	fetchedAt   time.Time
	lastAttempt time.Time
}

const (
	bangumiTTL        = 7 * 24 * time.Hour
	bangumiRetryAfter = 10 * time.Minute
)

var bgm bangumiCache

func (b *bangumiCache) shouldRefresh() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	now := time.Now()
	if now.Sub(b.fetchedAt) < bangumiTTL {
		return false
	}
	if now.Sub(b.lastAttempt) < bangumiRetryAfter {
		return false
	}
	return true
}

func (b *bangumiCache) snapshot() NewBangumi {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data
}

func (b *bangumiCache) storeResult(data NewBangumi, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.lastAttempt = now
	if ok {
		b.data = data
		b.fetchedAt = now
	}
}

func fetchBangumi(url string) (NewBangumi, error) {
	var zero NewBangumi
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("User-Agent", "trim21/bangumi-episode-ics (https://github.com/Trim21/bangumi-episode-calendar)")

	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return zero, echo.NewHTTPError(resp.StatusCode, "bangumi upstream error")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	var m NewBangumi
	if err := json.Unmarshal(body, &m); err != nil {
		return zero, err
	}
	sort.Slice(m.Data, func(i, j int) bool { return m.Data[i].Subject.ID < m.Data[j].Subject.ID })
	return m, nil
}

func Bangumi(c *echo.Context) error {
	if bgm.shouldRefresh() {
		cfg, err := store.ReadConfig()
		if err != nil {
			return err
		}
		url := "https://api.bgm.tv/v0/users/" + cfg.BangumiUserID + "/collections?subject_type=2&limit=100&offset=0"
		data, err := fetchBangumi(url)
		bgm.storeResult(data, err == nil)
	}
	return c.Render(200, "page-bangumi.template", bgm.snapshot())
}
