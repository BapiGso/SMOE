// Package migrate 把旧版的 usr/posts、usr/pages 一次性转换成 Hugo 站点结构：
// 文章与页面写到 usr/content，浏览量、点赞和评论写到 usr/data。
// 旧文件只读不改；新目录先在临时目录里生成，全部成功后才改名到位。
package migrate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BapiGso/SMOE/internal/moe/blogdata"

	"gopkg.in/yaml.v3"
)

// 旧版文件名里的 cid 是东八区的发布时间
var cst = time.FixedZone("CST", 8*3600)

// legacy 是旧版 .md 的 YAML 头
type legacy struct {
	Cover    string             `yaml:"cover"`
	Music    string             `yaml:"music"`
	Views    uint               `yaml:"views"`
	Likes    uint               `yaml:"likes"`
	Status   string             `yaml:"status"`
	Created  string             `yaml:"created"`
	Comments []blogdata.Comment `yaml:"comments"`
}

// front 是写给 Hugo 的 YAML 头，字段顺序即输出顺序
type front struct {
	Title  string `yaml:"title"`
	Slug   string `yaml:"slug,omitempty"`
	Date   string `yaml:"date,omitempty"`
	Draft  bool   `yaml:"draft,omitempty"`
	Layout string `yaml:"layout,omitempty"`
	Cover  string `yaml:"cover,omitempty"`
	Music  string `yaml:"music,omitempty"`
	Menus  string `yaml:"menus,omitempty"`
	Weight int    `yaml:"weight,omitempty"`
}

// Run 在 usr/content 和 usr/data 都不存在、且有旧数据时执行迁移，返回是否迁移过。
func Run(usrDir string) (bool, error) {
	contentDir := filepath.Join(usrDir, "content")
	dataDir := filepath.Join(usrDir, "data")
	if exists(contentDir) || exists(dataDir) {
		return false, nil
	}
	postsDir := filepath.Join(usrDir, "posts")
	pagesDir := filepath.Join(usrDir, "pages")
	if !exists(postsDir) && !exists(pagesDir) {
		return false, nil
	}

	tmp, err := os.MkdirTemp(usrDir, ".migrate-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)
	for _, d := range []string{"content/archives", "data"} {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.FromSlash(d)), 0o755); err != nil {
			return false, err
		}
	}
	if err := posts(postsDir, tmp); err != nil {
		return false, fmt.Errorf("posts: %w", err)
	}
	if err := pages(pagesDir, tmp); err != nil {
		return false, fmt.Errorf("pages: %w", err)
	}

	if err := os.Rename(filepath.Join(tmp, "data"), dataDir); err != nil {
		return false, err
	}
	if err := os.Rename(filepath.Join(tmp, "content"), contentDir); err != nil {
		os.RemoveAll(dataDir) // 刚放进去的，撤回后下次启动会重新迁移
		return false, err
	}
	return true, nil
}

func posts(dir, out string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	stats := map[string]blogdata.Stat{}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		idStr, title, _ := strings.Cut(name, "-")
		cid, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		fm, body, err := read(filepath.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		if title == "" {
			title = idStr
		}
		f := front{
			Title: title,
			Slug:  idStr,
			Date:  time.Unix(cid, 0).In(cst).Format(time.RFC3339),
			Draft: fm.Status != "" && fm.Status != "publish",
			Cover: fm.Cover,
			Music: fm.Music,
		}
		if err := write(filepath.Join(out, "content", "archives", e.Name()), f, body); err != nil {
			return err
		}
		stats[idStr] = blogdata.Stat{Views: fm.Views, Likes: fm.Likes}
		if len(fm.Comments) > 0 {
			b, err := yaml.Marshal(fm.Comments)
			if err != nil {
				return err
			}
			if err := blogdata.WriteFileAtomic(filepath.Join(out, "data", "comments", idStr+".yaml"), b); err != nil {
				return err
			}
		}
	}
	b, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "data", "stats.json"), b, 0o644)
}

// pages 转换独立页面：按文件名顺序进导航菜单；
// 时间线和追番原本是占位页（正文只有 [archives] / [bangumi]），改由主题模板渲染。
func pages(dir, out string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	weight := 0
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		fm, body, err := read(filepath.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		slug, title, _ := strings.Cut(name, "-")
		if title == "" {
			title = slug
		}
		weight++
		f := front{Title: title, Menus: "main", Weight: weight}
		target := filepath.Join(out, "content", e.Name())
		placeholder := strings.TrimSpace(string(body)) == "["+slug+"]"
		switch {
		case slug == "archives":
			// 文章所在的 section 本身就是时间线
			target = filepath.Join(out, "content", "archives", "_index.md")
		case slug == "bangumi":
			f.Slug, f.Layout = slug, "bangumi"
		default:
			f.Slug, f.Date = slug, fm.Created
		}
		if placeholder {
			body = nil
		}
		if err := write(target, f, body); err != nil {
			return err
		}
	}
	return nil
}

// read 拆出旧文件的 YAML 头和正文；正文去掉开头的空行（与旧版 ParseFile 一致），其余原样保留。
func read(name string) (legacy, []byte, error) {
	var fm legacy
	b, err := os.ReadFile(name)
	if err != nil {
		return fm, nil, err
	}
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	head, body, ok := split(b)
	if !ok {
		return fm, b, nil
	}
	if err := yaml.Unmarshal(head, &fm); err != nil {
		return fm, nil, err
	}
	return fm, bytes.TrimLeft(body, "\r\n"), nil
}

// split 识别以 --- 行开头、以 --- 行结束的头部。
func split(b []byte) (head, body []byte, ok bool) {
	rest, found := bytes.CutPrefix(b, []byte("---\n"))
	if !found {
		if rest, found = bytes.CutPrefix(b, []byte("---\r\n")); !found {
			return nil, b, false
		}
	}
	for off := 0; off < len(rest); {
		end := bytes.IndexByte(rest[off:], '\n')
		line := rest[off:]
		next := len(rest)
		if end >= 0 {
			line = rest[off : off+end]
			next = off + end + 1
		}
		if string(bytes.TrimRight(line, "\r")) == "---" {
			return rest[:off], rest[next:], true
		}
		off = next
	}
	return nil, b, false
}

func write(name string, f front, body []byte) error {
	head, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(head)
	buf.WriteString("---\n")
	if len(body) > 0 {
		buf.WriteString("\n")
		buf.Write(body)
	}
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, buf.Bytes(), 0o644)
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
