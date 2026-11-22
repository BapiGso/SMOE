package moe

import (
	"cmp"
	"mime"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// 带指纹的资源（resources.Fingerprint 生成的 name.<sha256>.css）内容永不变化，可以长期缓存
var fingerprinted = regexp.MustCompile(`\.[0-9a-f]{64}\.[a-z]+$`)

// front 从 Hugo 的构建产物里取前台页面。
// 地址不带尾部斜杠（/archives/123），对应产物 /archives/123/index.html；
// 请求头 Accept 含 text/markdown 时优先返回同目录的 index.md，供 AI 读取原文。
// 打开文章时记一次浏览（同一访客同一篇在去重周期内只算一次）。
func (s *Smoe) front(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		s.fail(w, http.StatusMethodNotAllowed)
		return
	}
	p := r.URL.Path
	// 去掉尾部斜杠后 301。ServeMux 已经把 // 之类的地址规范化了，这里不会跳去站外
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		u := *r.URL
		u.Path, u.RawPath = strings.TrimRight(p, "/"), ""
		http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
		return
	}
	if !s.site.Ready() { // 只有首次构建失败时才会这样
		w.Header().Set("Retry-After", "5")
		s.fail(w, http.StatusServiceUnavailable)
		return
	}
	if p == "/feed" {
		p = "/feed.xml"
	}
	if dir, id := path.Split(p); dir == "/archives/" && r.Method == http.MethodGet {
		if _, ok := s.site.Post(id); ok && s.viewSeen.allow(clientIP(r)+"\x00"+id) {
			s.data.View(id)
		}
	}

	candidates := []string{p}
	if path.Ext(p) == "" {
		w.Header().Add("Vary", "Accept")
		candidates = []string{path.Join(p, "index.html")}
		if strings.Contains(r.Header.Get("Accept"), "text/markdown") {
			candidates = append([]string{path.Join(p, "index.md")}, candidates...)
		}
	}
	for _, name := range candidates {
		f, fi, ok := s.site.Open(name)
		if !ok {
			continue
		}
		defer f.Close()
		h := w.Header()
		switch {
		case path.Ext(name) == ".md":
			h.Set("Content-Type", "text/markdown; charset=utf-8")
			h.Set("X-Markdown-Tokens", strconv.FormatInt(fi.Size()/4, 10)) // 粗略估算
		case name == "/feed.xml":
			h.Set("Content-Type", "application/rss+xml; charset=utf-8")
		}
		if fingerprinted.MatchString(name) {
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeContent(w, r, name, fi.ModTime(), f) // 其他类型按扩展名推断
		return
	}
	s.fail(w, http.StatusNotFound)
}

// uploads 提供 usr/uploads 下的文件，不列目录。
// 旁边有预先压好的 name.gz 且浏览器接受 gzip 时直接发压缩版（WASM 游戏等大文件）。
func (s *Smoe) uploads(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("path")
	w.Header().Add("Vary", "Accept-Encoding")
	f, err := (*os.File)(nil), error(nil)
	if r.Header.Get("Range") == "" && acceptsGzip(r.Header.Get("Accept-Encoding")) {
		if f, err = os.OpenInRoot("usr/uploads", name+".gz"); err == nil {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Type", cmp.Or(mime.TypeByExtension(path.Ext(name)), "application/octet-stream"))
		}
	}
	if f == nil {
		if f, err = os.OpenInRoot("usr/uploads", name); err != nil {
			s.fail(w, http.StatusNotFound)
			return
		}
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		w.Header().Del("Content-Encoding")
		s.fail(w, http.StatusNotFound)
		return
	}
	if w.Header().Get("Content-Encoding") != "" { // 设了编码时 ServeContent 不写长度
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	}
	http.ServeContent(w, r, name, fi.ModTime(), f)
}

// acceptsGzip 看 Accept-Encoding 里有没有 gzip（q=0 视为拒绝）。
func acceptsGzip(header string) bool {
	for part := range strings.SplitSeq(header, ",") {
		coding, params, _ := strings.Cut(part, ";")
		if strings.EqualFold(strings.TrimSpace(coding), "gzip") {
			q, ok := strings.CutPrefix(strings.ReplaceAll(params, " ", ""), "q=")
			return !ok || strings.Trim(q, "0.") != ""
		}
	}
	return false
}
