package blogdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	bangumiTTL        = 7 * 24 * time.Hour
	bangumiRetryAfter = 10 * time.Minute
)

// RunBangumi 让 data/bangumi.json 保持新鲜（超过 7 天就重新拉取，失败 10 分钟后重试，旧数据保留），
// 直到 ctx 结束。每次写入新数据后调用 updated。
func RunBangumi(ctx context.Context, dataDir, userID string, updated func()) {
	if userID == "" {
		return
	}
	name := filepath.Join(dataDir, "bangumi.json")
	for {
		wait := bangumiTTL
		if fi, err := os.Stat(name); err == nil && time.Since(fi.ModTime()) < bangumiTTL {
			wait = bangumiTTL - time.Since(fi.ModTime())
		} else if err := fetchBangumi(ctx, userID, name); err != nil {
			slog.Warn("bangumi: fetch failed, keeping old data", "err", err)
			wait = bangumiRetryAfter
		} else if updated != nil {
			updated()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// fetchBangumi 拉取用户的动画收藏并原样保存，只把条目按 subject.id 升序排好。
func fetchBangumi(ctx context.Context, userID, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	url := "https://api.bgm.tv/v0/users/" + userID + "/collections?subject_type=2&limit=100&offset=0"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "trim21/bangumi-episode-ics (https://github.com/Trim21/bangumi-episode-calendar)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return err
	}
	var items []json.RawMessage
	if err := json.Unmarshal(doc["data"], &items); err != nil {
		return fmt.Errorf("data: %w", err)
	}
	ids := make([]int, len(items))
	for i, it := range items {
		var v struct {
			Subject struct {
				ID int `json:"id"`
			} `json:"subject"`
		}
		_ = json.Unmarshal(it, &v)
		ids[i] = v.Subject.ID
	}
	sort.Stable(byID{items, ids})
	if doc["data"], err = json.Marshal(items); err != nil {
		return err
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return WriteFileAtomic(name, out)
}

type byID struct {
	items []json.RawMessage
	ids   []int
}

func (s byID) Len() int           { return len(s.items) }
func (s byID) Less(i, j int) bool { return s.ids[i] < s.ids[j] }
func (s byID) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
	s.ids[i], s.ids[j] = s.ids[j], s.ids[i]
}
