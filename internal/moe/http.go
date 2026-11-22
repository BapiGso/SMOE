package moe

import (
	"cmp"
	"crypto/sha256"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/webdav"
)

func (s *Smoe) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.front) // 首页、分页、文章、独立页面、时间线、追番、RSS、css/js 都来自 Hugo 构建产物
	mux.HandleFunc("POST /archives/{cid}/comment", s.submitComment)
	mux.HandleFunc("POST /archives/{cid}/like", s.like)
	mux.HandleFunc("GET /comment/moderate", s.moderate) // 邮件里的审核链接，GET 只显示确认页
	mux.HandleFunc("POST /comment/moderate", s.moderate)
	mux.HandleFunc("GET /usr/uploads/{path...}", s.uploads)
	mux.HandleFunc("/webdav", serveWebDAV) // 写文章、传图片、改配置都走 WebDAV
	mux.HandleFunc("/webdav/", serveWebDAV)
	// 跨站来的 POST（评论、点赞、审核）一律拒绝；WebDAV 客户端不带 Origin/Sec-Fetch-Site，不受影响
	return logRequests(secureHeaders(http.NewCrossOriginProtection().Handler(mux)))
}

// fail 写错误响应：404 用 Hugo 构建出的 404.html，其他只回状态文本。
func (s *Smoe) fail(w http.ResponseWriter, code int) {
	if code == http.StatusNotFound {
		if f, _, ok := s.site.Open("/404.html"); ok {
			defer f.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(code)
			_, _ = io.Copy(w, f)
			return
		}
	}
	http.Error(w, http.StatusText(code), code)
}

// clientIP 是连接的对端地址，评论、点赞限流和浏览去重按它区分访客。
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// cooldown 记下每个键上次放行的时间，window 内同一个键只放行一次。
type cooldown struct {
	window time.Duration

	mu    sync.Mutex
	last  map[string]time.Time
	swept time.Time
}

func (c *cooldown) allow(key string) bool {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if now.Sub(c.swept) > c.window { // 每个周期清一次过期的键，表不会无限增长
		for k, t := range c.last {
			if now.Sub(t) >= c.window {
				delete(c.last, k)
			}
		}
		c.swept = now
	}
	if t, ok := c.last[key]; ok && now.Sub(t) < c.window {
		return false
	}
	if c.last == nil {
		c.last = map[string]time.Time{}
	}
	c.last[key] = now
	return true
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.code == 0 {
		w.code = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		slog.Info("http", "method", r.Method, "ip", clientIP(r), "status", cmp.Or(sw.code, http.StatusOK), "url", r.RequestURI)
	})
}

var davHandler = &webdav.Handler{
	Prefix:     "/webdav",
	FileSystem: webdav.Dir("usr"),
	LockSystem: webdav.NewMemLS(),
}

func serveWebDAV(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || !davAuth(user, pass) {
		w.Header().Set("WWW-Authenticate", `Basic realm="SMOE"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	// no-transform: 禁止 CDN/代理压缩或修改响应体（WebDAV XML/二进制会被破坏）；private: 不在 CDN 层缓存
	w.Header().Set("Cache-Control", "no-transform, private")
	davHandler.ServeHTTP(w, r)
}

// davPassed 是上次认证通过的 用户名+密码+密码哈希 的摘要。
// WebDAV 客户端每个文件都要发请求，不能每次都跑 bcrypt；哈希在摘要里，改了密码立即生效。
var davPassed atomic.Pointer[[sha256.Size]byte]

func davAuth(user, pass string) bool {
	cfg, err := readConfig()
	if err != nil || cfg.Name == "" || cfg.Name != user {
		return false
	}
	sum := sha256.Sum256([]byte(user + "\x00" + pass + "\x00" + cfg.Password))
	if p := davPassed.Load(); p != nil && *p == sum {
		return true
	}
	if bcrypt.CompareHashAndPassword([]byte(cfg.Password), []byte(pass)) != nil {
		return false
	}
	davPassed.Store(&sum)
	return true
}
