package moe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/BapiGso/SMOE/internal/moe/engine"

	"golang.org/x/crypto/bcrypt"
)

func TestCooldown(t *testing.T) {
	c := &cooldown{window: time.Hour}
	if !c.allow("a") || c.allow("a") {
		t.Fatal("same key should pass once per window")
	}
	if !c.allow("b") {
		t.Fatal("other keys are independent")
	}
	c.last["a"] = time.Now().Add(-2 * time.Hour)
	if !c.allow("a") {
		t.Fatal("key should pass again after the window")
	}
}

func TestParseComment(t *testing.T) {
	valid := url.Values{"author": {"晓梦"}, "mail": {"a@example.com"}, "text": {"你好"}, "url": {"https://smoe.cc"}, "parent": {"3"}}
	cases := []struct {
		name string
		edit func(url.Values)
		ok   bool
	}{
		{"valid", func(url.Values) {}, true},
		{"no_url_no_parent", func(v url.Values) { v.Del("url"); v.Set("parent", "0") }, true},
		{"empty_author", func(v url.Values) { v.Set("author", "") }, false},
		{"author_51_runes", func(v url.Values) { v.Set("author", strings.Repeat("字", 51)) }, false},
		{"author_50_runes", func(v url.Values) { v.Set("author", strings.Repeat("字", 50)) }, true},
		{"bad_mail", func(v url.Values) { v.Set("mail", "not-a-mail") }, false},
		{"mail_with_name", func(v url.Values) { v.Set("mail", "X <a@example.com>") }, false},
		{"empty_text", func(v url.Values) { v.Set("text", "") }, false},
		{"javascript_url", func(v url.Values) { v.Set("url", "javascript:alert(1)") }, false},
		{"relative_url", func(v url.Values) { v.Set("url", "/x") }, false},
		{"bad_parent", func(v url.Values) { v.Set("parent", "x") }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := url.Values{}
			for k, vs := range valid {
				v[k] = append([]string(nil), vs...)
			}
			tc.edit(v)
			r := httptest.NewRequest(http.MethodPost, "/archives/1/comment", strings.NewReader(v.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if _, ok := parseComment(r); ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
		})
	}
}

func TestRoutes(t *testing.T) {
	const user, pass = "testuser", "testpass"
	dir := t.TempDir()
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "usr", "uploads", "background"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf("name: %s\npassword: '%s'\n", user, hash)
	if err := os.WriteFile(filepath.Join(dir, "usr", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "usr", "uploads", "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	// 没有构建过的引擎：404 回纯文本，前台回 503
	site, err := engine.New("usr", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { site.Close() })
	h := (&Smoe{site: site}).routes()
	do := func(method, target string, edit func(*http.Request)) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, target, nil)
		if edit != nil {
			edit(r)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	t.Run("webdav_needs_auth", func(t *testing.T) {
		w := do("PROPFIND", "/webdav/", nil)
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("got %d", w.Code)
		}
		w = do("PROPFIND", "/webdav/", func(r *http.Request) { r.SetBasicAuth(user, "wrong") })
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("wrong password: got %d", w.Code)
		}
	})
	t.Run("webdav_propfind", func(t *testing.T) {
		for range 2 { // 第二次走认证缓存
			w := do("PROPFIND", "/webdav/", func(r *http.Request) { r.SetBasicAuth(user, pass); r.Header.Set("Depth", "1") })
			if w.Code != http.StatusMultiStatus || w.Header().Get("Cache-Control") == "" {
				t.Fatalf("got %d", w.Code)
			}
		}
	})
	t.Run("cross_site_post_rejected", func(t *testing.T) {
		w := do(http.MethodPost, "http://example.com/archives/1/like", func(r *http.Request) {
			r.Header.Set("Sec-Fetch-Site", "cross-site")
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("got %d", w.Code)
		}
	})
	t.Run("trailing_slash", func(t *testing.T) {
		w := do(http.MethodGet, "/archives/1/?a=b", nil)
		if w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != "/archives/1?a=b" {
			t.Fatalf("got %d %q", w.Code, w.Header().Get("Location"))
		}
		w = do(http.MethodGet, "//evil.com/", nil)
		if loc := w.Header().Get("Location"); strings.HasPrefix(loc, "//") {
			t.Fatalf("open redirect to %q", loc)
		}
	})
	t.Run("uploads", func(t *testing.T) {
		if w := do(http.MethodGet, "/usr/uploads/a.txt", nil); w.Code != http.StatusOK || w.Body.String() != "hi" {
			t.Fatalf("file: got %d", w.Code)
		}
		if w := do(http.MethodGet, "/usr/uploads/background", nil); w.Code != http.StatusNotFound {
			t.Fatalf("dir should not be listed: got %d", w.Code)
		}
	})
	t.Run("not_built_yet", func(t *testing.T) {
		if w := do(http.MethodGet, "/", nil); w.Code != http.StatusServiceUnavailable || w.Header().Get("Retry-After") == "" {
			t.Fatalf("got %d", w.Code)
		}
		if w := do(http.MethodPut, "/", nil); w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("PUT: got %d", w.Code)
		}
	})
	t.Run("security_headers", func(t *testing.T) {
		w := do(http.MethodGet, "/usr/uploads/a.txt", func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https") })
		if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Strict-Transport-Security") == "" {
			t.Fatalf("headers: %v", w.Header())
		}
	})
}
