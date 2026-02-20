package mymiddleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"

	"SMOE/moe/mymiddleware"
)

// setupWebDAVEnv creates a temp directory with usr/config.yaml and
// changes the process working directory to it for the duration of the test.
func setupWebDAVEnv(t *testing.T, username, password string) {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "usr"), 0755); err != nil {
		t.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	// bcrypt hash 含 $ 等特殊字符，用单引号包裹以免 YAML 误解析
	config := fmt.Sprintf("user:\n  name: %s\n  password: '%s'\n", username, hash)
	if err := os.WriteFile(filepath.Join(tmpDir, "usr", "config.yaml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
}

func TestWebDAV(t *testing.T) {
	const testUser, testPass = "testuser", "testpass"
	setupWebDAVEnv(t, testUser, testPass)

	e := echo.New()
	e.Pre(mymiddleware.WebDAV())
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)

	t.Run("non_webdav_path_passes_through", func(t *testing.T) {
		// 非 /webdav 路径不应触发 Basic Auth
		resp, err := http.Get(ts.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			t.Error("非 WebDAV 路径不应要求认证")
		}
	})

	t.Run("no_auth_returns_401", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/webdav/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("无认证：期望 401，得到 %d", resp.StatusCode)
		}
		if resp.Header.Get("WWW-Authenticate") == "" {
			t.Error("缺少 WWW-Authenticate 响应头")
		}
	})

	t.Run("wrong_password_returns_401", func(t *testing.T) {
		req, _ := http.NewRequest("PROPFIND", ts.URL+"/webdav/", nil)
		req.SetBasicAuth(testUser, "wrongpass")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("密码错误：期望 401，得到 %d", resp.StatusCode)
		}
	})

	t.Run("propfind_root_returns_207", func(t *testing.T) {
		req, _ := http.NewRequest("PROPFIND", ts.URL+"/webdav/", nil)
		req.SetBasicAuth(testUser, testPass)
		req.Header.Set("Depth", "1")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMultiStatus {
			t.Errorf("PROPFIND 连通性：期望 207，得到 %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct == "" {
			t.Error("缺少 Content-Type 响应头")
		}
	})

	t.Run("cache_control_no_transform", func(t *testing.T) {
		req, _ := http.NewRequest("PROPFIND", ts.URL+"/webdav/", nil)
		req.SetBasicAuth(testUser, testPass)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		cc := resp.Header.Get("Cache-Control")
		if cc == "" {
			t.Error("缺少 Cache-Control 响应头")
		}
	})
}
