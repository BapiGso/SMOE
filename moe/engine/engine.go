// Package engine 把 Hugo 当作库嵌入：usr/ 就是 Hugo 站点目录，构建结果放在内存里，前台直接读取。
//
// 约定：content/ 只由用户写（WebDAV 或本地编辑器），data/ 只由程序写（浏览量、评论、番剧）。
// 任何改动都触发一次全量重建，构建到一份新的内存文件系统，成功后才替换；
// 失败时继续提供上一次的结果，所以写坏一篇文章不会让站点下线。
package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bep/logg"
	"github.com/fsnotify/fsnotify"
	"github.com/gohugoio/hugo/config"
	"github.com/gohugoio/hugo/config/allconfig"
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/hugofs"
	"github.com/gohugoio/hugo/hugolib"
	"github.com/spf13/afero"
)

const themeName = "smoe"

// Engine 持有最近一次成功构建的产物。
type Engine struct {
	usrDir   string // Hugo 的 workingDir
	themeDir string // 含 site.toml 和 smoe/ 主题目录
	workDir  string // 临时目录：解压的主题、Hugo 的 resources 与缓存

	buildMu sync.Mutex // 同一时间只跑一次构建

	mu   sync.RWMutex
	snap *snapshot

	kick chan struct{}
}

// snapshot 是一次构建的结果，生成后不再修改。
type snapshot struct {
	out     afero.Fs          // 发布目录，即整个网站
	posts   map[string]string // archives 下已发布文章：ID → 标题
	baseURL string            // 不带尾部斜杠
}

// New 解压内嵌主题并准备好引擎，不做构建；调用方随后调用 Build。
// 设置环境变量 SMOE_THEME_DIR 可直接使用磁盘上的主题目录（开发时改模板免重编译）。
func New(usrDir string, theme fs.FS) (*Engine, error) {
	usrDir, err := filepath.Abs(usrDir)
	if err != nil {
		return nil, err
	}
	workDir, err := os.MkdirTemp("", "smoe-hugo-")
	if err != nil {
		return nil, err
	}
	themeDir := os.Getenv("SMOE_THEME_DIR")
	if themeDir == "" {
		themeDir = filepath.Join(workDir, "theme")
		if err := extract(theme, themeDir); err != nil {
			os.RemoveAll(workDir)
			return nil, err
		}
	} else if themeDir, err = filepath.Abs(themeDir); err != nil {
		os.RemoveAll(workDir)
		return nil, err
	}
	return &Engine{
		usrDir:   usrDir,
		themeDir: themeDir,
		workDir:  workDir,
		kick:     make(chan struct{}, 1),
	}, nil
}

// Build 全量构建一次，成功后替换当前产物。
func (e *Engine) Build() error {
	e.buildMu.Lock()
	defer e.buildMu.Unlock()

	start := time.Now()
	snap, err := e.build()
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.snap = snap
	e.mu.Unlock()
	slog.Info("hugo build", "posts", len(snap.posts), "took", time.Since(start).Round(time.Millisecond))
	return nil
}

func (e *Engine) build() (*snapshot, error) {
	// 这些由程序决定，用户配置覆盖不了
	flags := config.New()
	flags.Set("workingDir", e.usrDir)
	flags.Set("themesDir", e.themeDir)
	flags.Set("theme", themeName)
	flags.Set("noBuildLock", true)
	flags.Set("resourceDir", filepath.Join(e.workDir, "resources"))
	flags.Set("cacheDir", filepath.Join(e.workDir, "cache"))

	// 主题自带的基础配置在前，usr/hugo.toml（可选）在后，后者覆盖前者
	configs, err := allconfig.LoadConfig(allconfig.ConfigSourceDescriptor{
		Flags:    flags,
		Fs:       hugofs.Os,
		Filename: filepath.Join(e.themeDir, "site.toml") + "," + filepath.Join(e.usrDir, "hugo.toml"),
	})
	if err != nil {
		return nil, err
	}

	// 与 hugo server --renderToMemory 相同的设置
	flags.Set("publishDir", configs.Base.PublishDir)
	flags.Set("publishDirDynamic", "/")
	flags.Set("publishDirStatic", "/")
	fs := hugofs.NewFromSourceAndDestination(hugofs.Os, afero.NewMemMapFs(), flags)

	h, err := hugolib.NewHugoSites(deps.DepsCfg{
		Configs:  configs,
		Fs:       fs,
		StdOut:   os.Stdout,
		StdErr:   os.Stderr,
		LogLevel: logg.LevelWarn,
	})
	if err != nil {
		return nil, err
	}
	// 产物写进了 fs.PublishDir，Hugo 本身用完即关
	defer h.Close()
	if err := h.Build(hugolib.BuildCfg{}); err != nil {
		return nil, err
	}
	if err := copyStatic(h, fs.PublishDirStatic); err != nil {
		return nil, err
	}

	snap := &snapshot{
		out:     fs.PublishDir,
		posts:   map[string]string{},
		baseURL: strings.TrimSuffix(configs.Base.BaseURL, "/"),
	}
	// 文章以地址最后一段为 ID（/archives/<id>），与模板里 path.Base .RelPermalink 一致；
	// 旧文章的 slug 就是 cid，新文章不写 slug 时是文件名
	for _, p := range h.Sites[0].RegularPages() {
		if p.Section() == "archives" {
			snap.posts[path.Base(p.RelPermalink())] = p.Title()
		}
	}
	return snap, nil
}

// copyStatic 把 static/（主题与 usr/ 的合并视图）复制进发布目录。
// Hugo 命令行是在构建之外单独做这一步的（commands.copyStatic），作为库调用时要自己做。
// 产物在内存里，大文件应放 usr/uploads 而不是 usr/static。
func copyStatic(h *hugolib.HugoSites, dst afero.Fs) error {
	root := string(filepath.Separator)
	for _, sfs := range h.BaseFs.SourceFilesystems.Static {
		err := afero.Walk(sfs.Fs, root, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				if p == root && errors.Is(err, fs.ErrNotExist) {
					return nil // 没有 static 目录
				}
				return err
			}
			if fi.IsDir() {
				return nil
			}
			b, err := afero.ReadFile(sfs.Fs, p)
			if err != nil {
				return err
			}
			return afero.WriteFile(dst, filepath.Join(root, sfs.PublishFolder, p), b, 0o644)
		})
		if err != nil {
			return fmt.Errorf("copy static: %w", err)
		}
	}
	return nil
}

// Post 按 ID（/archives/<id> 的最后一段）查找已发布的文章，返回标题。评论、点赞、浏览只接受存在的文章。
func (e *Engine) Post(id string) (string, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.snap == nil {
		return "", false
	}
	title, ok := e.snap.posts[id]
	return title, ok
}

// BaseURL 返回配置里的站点地址（不带尾部斜杠），用于邮件里的链接。
func (e *Engine) BaseURL() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.snap == nil {
		return ""
	}
	return e.snap.baseURL
}

// Trigger 请求一次异步重建；短时间内的多次请求会合并成一次。程序写完 data/ 后调用。
func (e *Engine) Trigger() {
	select {
	case e.kick <- struct{}{}:
	default:
	}
}

// 会影响构建结果的 usr/ 子目录，递归监听
var watchedDirs = []string{"content", "data", "layouts", "assets", "static", "i18n", "archetypes"}

// 改动停止这么久之后才重建，编辑器保存时的连串事件只算一次
const settle = 500 * time.Millisecond

// Run 监听 usr/ 的改动并在需要时重建，直到 ctx 结束。
// 监听器起不来（比如 inotify 句柄耗尽）时只记录日志，Trigger 仍然有效。
func (e *Engine) Run(ctx context.Context) {
	var events <-chan fsnotify.Event
	var errs <-chan error
	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("engine: file watcher unavailable, only program writes trigger rebuilds", "err", err)
	} else {
		defer w.Close()
		e.watch(w)
		events, errs = w.Events, w.Errors
	}

	timer := time.NewTimer(settle)
	timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if !e.relevant(ev) {
				continue
			}
			// 监听不递归，新建的目录要补上
			if ev.Has(fsnotify.Create) {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					addTree(w, ev.Name)
				}
			}
			timer.Reset(settle)
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			slog.Warn("engine: watcher", "err", err)
		case <-e.kick:
			timer.Reset(settle)
		case <-timer.C:
			if err := e.Build(); err != nil {
				slog.Error("hugo build failed, still serving the previous build", "err", err)
			}
		}
	}
}

func (e *Engine) watch(w *fsnotify.Watcher) {
	// usr/ 本身只为了 hugo.toml 和上面那些目录被新建
	if err := w.Add(e.usrDir); err != nil {
		slog.Warn("engine: watch", "dir", e.usrDir, "err", err)
	}
	for _, d := range watchedDirs {
		addTree(w, filepath.Join(e.usrDir, d))
	}
	// 自动封面列表来自这个目录
	_ = w.Add(filepath.Join(e.usrDir, "uploads", "background"))
	if os.Getenv("SMOE_THEME_DIR") != "" {
		addTree(w, e.themeDir)
	}
}

// relevant 过滤掉与构建无关的事件：权限变化、编辑器临时文件、usr/ 下的其他文件。
func (e *Engine) relevant(ev fsnotify.Event) bool {
	if ev.Op == fsnotify.Chmod {
		return false
	}
	base := filepath.Base(ev.Name)
	if strings.HasSuffix(base, "~") || strings.HasPrefix(base, "#") || strings.HasPrefix(base, "._") ||
		base == ".DS_Store" || strings.HasSuffix(base, ".swp") || strings.HasSuffix(base, ".swx") || strings.HasSuffix(base, ".tmp") {
		return false
	}
	if filepath.Dir(ev.Name) == e.usrDir {
		return base == "hugo.toml" || slices.Contains(watchedDirs, base)
	}
	return true
}

// addTree 递归监听 dir 及其所有子目录；目录不存在时什么也不做。
func addTree(w *fsnotify.Watcher, dir string) {
	if w == nil {
		return
	}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if err := w.Add(p); err != nil {
				slog.Warn("engine: watch", "dir", p, "err", err)
			}
		}
		return nil
	})
}

// File 返回最近一次成功构建中某个文件的内容，name 形如 "/archives/1586584980/index.html"。
func (e *Engine) File(name string) ([]byte, time.Time, bool) {
	e.mu.RLock()
	snap := e.snap
	e.mu.RUnlock()
	if snap == nil {
		return nil, time.Time{}, false
	}
	out := snap.out
	fi, err := out.Stat(name)
	if err != nil || fi.IsDir() {
		return nil, time.Time{}, false
	}
	b, err := afero.ReadFile(out, name)
	if err != nil {
		return nil, time.Time{}, false
	}
	return b, fi.ModTime(), true
}

// Ready 报告是否已有可用的构建产物。
func (e *Engine) Ready() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.snap != nil
}

// Close 删除临时目录。调用前先结束 Run。
func (e *Engine) Close() error {
	e.buildMu.Lock()
	defer e.buildMu.Unlock()
	return os.RemoveAll(e.workDir)
}

// extract 把内嵌主题写到磁盘，Hugo 只能从操作系统文件系统读主题。
func extract(src fs.FS, dst string) error {
	if src == nil {
		return errors.New("engine: no theme")
	}
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}
