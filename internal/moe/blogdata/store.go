package blogdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Store 管理 data/stats.json 和 data/comments/。
// 计数先在内存里累加，Flush 时才写盘，所以页面上的数字最多滞后一个刷新周期；评论立即写盘。
type Store struct {
	dir     string
	changed func() // 写入了会影响页面的数据，通常是 engine.Trigger

	mu    sync.Mutex
	stats map[string]Stat
	dirty bool

	commentMu sync.Mutex
}

// Open 读取 dir（即 usr/data）下已有的计数。
func Open(dir string, changed func()) (*Store, error) {
	s := &Store{dir: dir, changed: changed, stats: map[string]Stat{}}
	b, err := os.ReadFile(filepath.Join(dir, "stats.json"))
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &s.stats); err != nil {
			return nil, fmt.Errorf("stats.json: %w", err)
		}
	case !errors.Is(err, os.ErrNotExist):
		return nil, err
	}
	return s, nil
}

// View 记一次浏览。去重由调用方负责。
func (s *Store) View(slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.stats[slug]
	st.Views++
	s.stats[slug] = st
	s.dirty = true
}

// Like 记一次点赞。防刷由调用方负责。
func (s *Store) Like(slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.stats[slug]
	st.Likes++
	s.stats[slug] = st
	s.dirty = true
}

// Flush 把有变化的计数写进 stats.json。不要并发调用：平时只由 Run 调用，退出时 Run 结束后再调用一次。
func (s *Store) Flush() error {
	s.mu.Lock()
	if !s.dirty {
		s.mu.Unlock()
		return nil
	}
	b, err := json.MarshalIndent(s.stats, "", "  ")
	s.dirty = false
	s.mu.Unlock()
	if err == nil {
		err = WriteFileAtomic(filepath.Join(s.dir, "stats.json"), b)
	}
	if err != nil {
		s.mu.Lock()
		s.dirty = true
		s.mu.Unlock()
		return err
	}
	s.notify()
	return nil
}

// Run 每隔 every 写一次计数，直到 ctx 结束。最后一次由调用方在连接排空后 Flush。
func (s *Store) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.Flush(); err != nil {
				slog.Error("blogdata: flush failed", "err", err)
			}
		}
	}
}

// ErrNoComment 表示评论（或被回复的评论）不存在。
var ErrNoComment = errors.New("comment not found")

// AddComment 给文章追加一条待审核评论，返回保存后的评论和被回复的评论（如有）。
// 只能回复已通过的评论。内容原样保存，HTML 转义由模板负责。
func (s *Store) AddComment(slug string, c Comment) (Comment, *Comment, error) {
	s.commentMu.Lock()
	defer s.commentMu.Unlock()

	cs, err := s.comments(slug)
	if err != nil {
		return Comment{}, nil, err
	}
	var maxID uint
	var parent *Comment
	for i := range cs {
		maxID = max(maxID, cs[i].ID)
		if c.Parent != 0 && cs[i].ID == c.Parent && cs[i].Status == "approved" {
			p := cs[i]
			parent = &p
		}
	}
	if c.Parent != 0 && parent == nil {
		return Comment{}, nil, ErrNoComment
	}
	c.ID = maxID + 1
	c.Created = time.Now().Format(time.RFC3339)
	c.Status = "waiting"
	if err := s.saveComments(slug, append(cs, c)); err != nil {
		return Comment{}, nil, err
	}
	// 待审核的评论不显示，不必重建
	return c, parent, nil
}

// Comment 读取一条评论。
func (s *Store) Comment(slug string, id uint) (Comment, error) {
	s.commentMu.Lock()
	defer s.commentMu.Unlock()
	cs, err := s.comments(slug)
	if err != nil {
		return Comment{}, err
	}
	for _, c := range cs {
		if c.ID == id {
			return c, nil
		}
	}
	return Comment{}, ErrNoComment
}

// SetCommentStatus 修改评论状态（approved / spam / waiting），用于审核。
func (s *Store) SetCommentStatus(slug string, id uint, status string) (Comment, error) {
	s.commentMu.Lock()
	defer s.commentMu.Unlock()
	cs, err := s.comments(slug)
	if err != nil {
		return Comment{}, err
	}
	for i := range cs {
		if cs[i].ID != id {
			continue
		}
		if cs[i].Status == status {
			return cs[i], nil
		}
		cs[i].Status = status
		if err := s.saveComments(slug, cs); err != nil {
			return Comment{}, err
		}
		s.notify()
		return cs[i], nil
	}
	return Comment{}, ErrNoComment
}

func (s *Store) commentsPath(slug string) (string, error) {
	if slug == "" || slug == "." || slug == ".." || strings.ContainsAny(slug, `/\:`) {
		return "", fmt.Errorf("invalid slug %q", slug)
	}
	return filepath.Join(s.dir, "comments", slug+".yaml"), nil
}

func (s *Store) comments(slug string) ([]Comment, error) {
	name, err := s.commentsPath(slug)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cs []Comment
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, fmt.Errorf("comments/%s.yaml: %w", slug, err)
	}
	return cs, nil
}

func (s *Store) saveComments(slug string, cs []Comment) error {
	name, err := s.commentsPath(slug)
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(cs)
	if err != nil {
		return err
	}
	return WriteFileAtomic(name, b)
}

func (s *Store) notify() {
	if s.changed != nil {
		s.changed()
	}
}
