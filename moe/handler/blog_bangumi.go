package handler

import (
	"SMOE/moe/store"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
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
	lastErr     error
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

func (b *bangumiCache) snapshot() (NewBangumi, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data, !b.fetchedAt.IsZero()
}

func (b *bangumiCache) lastError() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastErr
}

func (b *bangumiCache) storeResult(data NewBangumi, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.lastAttempt = now
	if err == nil {
		b.data = data
		b.fetchedAt = now
		b.lastErr = nil
	} else {
		b.lastErr = echo.NewHTTPError(http.StatusBadGateway, "fetch bangumi collections failed").Wrap(err)
	}
}

func fetchBangumi(url string, appID string, token string) (NewBangumi, error) {
	var zero NewBangumi
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return zero, err
	}
	userAgent := "SMOE"
	if appID = strings.TrimSpace(appID); appID != "" {
		userAgent += "/" + appID
	}
	req.Header.Set("User-Agent", userAgent)
	if token = strings.TrimSpace(token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := fmt.Sprintf("bangumi upstream error: status=%d", resp.StatusCode)
		if len(body) > 0 {
			msg += ", body=" + strings.TrimSpace(string(body))
		}
		return zero, echo.NewHTTPError(resp.StatusCode, msg)
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
		userID := strings.TrimSpace(cfg.BangumiUserID)
		if userID == "" {
			return echo.NewHTTPError(http.StatusInternalServerError, "bangumi user id is empty")
		}
		endpoint := "https://api.bgm.tv/v0/users/" + url.PathEscape(userID) + "/collections?subject_type=2&limit=100&offset=0"
		data, err := fetchBangumi(endpoint, cfg.BangumiAppID, cfg.BgmToken)
		bgm.storeResult(data, err)
		if err != nil {
			slog.Error("fetch bangumi collections failed", "userID", userID, "err", err)
			if _, ok := bgm.snapshot(); !ok {
				return echo.NewHTTPError(http.StatusBadGateway, "fetch bangumi collections failed").Wrap(err)
			}
		}
	}
	data, ok := bgm.snapshot()
	if !ok {
		if err := bgm.lastError(); err != nil {
			return err
		}
		return echo.NewHTTPError(http.StatusServiceUnavailable, "bangumi data is unavailable")
	}
	return c.Render(200, "page-bangumi.template", data)
}
