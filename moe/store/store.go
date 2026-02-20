package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	postsDir   = "usr/posts"
	pagesDir   = "usr/pages"
	configFile = "usr/config.yaml"
)

// --- concurrency ---

var fileMutexes sync.Map

func getFileMutex(path string) *sync.Mutex {
	v, _ := fileMutexes.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// --- core helpers ---

// fmStatus returns the effective status of a post (defaults to "publish").
func fmStatus(fm FrontMatter) string {
	if fm.Status == "" {
		return "publish"
	}
	return fm.Status
}

// walkPosts calls fn for every valid post in usr/posts/*.md.
// Files whose names are not integers (e.g. drafts) are silently skipped.
func walkPosts(fn func(cid int, fm FrontMatter, body string)) error {
	entries, err := os.ReadDir(postsDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		cid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		fm, body, err := ParseFile(filepath.Join(postsDir, e.Name()))
		if err != nil {
			continue
		}
		fn(cid, fm, body)
	}
	return nil
}

// toComment converts a single FMComment to a Comments value.
func toComment(fc FMComment, cid uint) Comments {
	var url *string
	if fc.Url != "" {
		u := fc.Url
		url = &u
	}
	return Comments{
		Coid:    fc.ID,
		Cid:     cid,
		Created: parseRFC3339(fc.Created),
		Author:  fc.Author,
		Mail:    fc.Mail,
		Url:     url,
		Text:    fc.Content,
		Status:  fc.Status,
		Parent:  fc.Parent,
	}
}

// --- public API ---

// GetPostsByCidDesc returns paginated published posts sorted newest first.
func GetPostsByCidDesc(limit, offset int) ([]Contents, bool, error) {
	var all []Contents
	if err := walkPosts(func(cid int, fm FrontMatter, body string) {
		if fmStatus(fm) == "publish" {
			all = append(all, ToContents(fm, body, "post", cid))
		}
	}); err != nil {
		return nil, false, err
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Cid > all[j].Cid })

	if offset >= len(all) {
		return nil, false, nil
	}
	end := min(offset+limit+1, len(all))
	slice := all[offset:end]
	hasMore := len(slice) > limit
	if hasMore {
		slice = slice[:limit]
	}
	return slice, hasMore, nil
}

// GetAllPublishedPosts returns all published posts sorted newest first.
func GetAllPublishedPosts() ([]Contents, error) {
	var result []Contents
	if err := walkPosts(func(cid int, fm FrontMatter, body string) {
		if fmStatus(fm) == "publish" {
			result = append(result, ToContents(fm, body, "post", cid))
		}
	}); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Cid > result[j].Cid })
	return result, nil
}

// GetAllPages returns all pages from usr/pages/.
func GetAllPages() ([]Contents, error) {
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		return nil, err
	}
	var pages []Contents
	for _, e := range entries {
		slug, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		fm, body, err := ParseFile(filepath.Join(pagesDir, e.Name()))
		if err != nil {
			continue
		}
		c := ToContents(fm, body, "page", 0)
		c.Slug = slug
		pages = append(pages, c)
	}
	return pages, nil
}

// GetPostByCid returns a published post and its approved comments.
func GetPostByCid(cid int) (Contents, []Comments, error) {
	path := filepath.Join(postsDir, strconv.Itoa(cid)+".md")
	fm, body, err := ParseFile(path)
	if err != nil {
		return Contents{}, nil, err
	}
	if fmStatus(fm) != "publish" {
		return Contents{}, nil, fmt.Errorf("not found")
	}
	var approved []Comments
	for _, fc := range fm.Comments {
		if fc.Status == "approved" {
			approved = append(approved, toComment(fc, uint(cid)))
		}
	}
	sort.Slice(approved, func(i, j int) bool { return approved[i].Created < approved[j].Created })
	return ToContents(fm, body, "post", cid), approved, nil
}

// GetPageBySlug returns a page by its slug (filename without .md).
func GetPageBySlug(slug string) (Contents, error) {
	fm, body, err := ParseFile(filepath.Join(pagesDir, slug+".md"))
	if err != nil {
		return Contents{}, err
	}
	c := ToContents(fm, body, "page", 0)
	c.Slug = slug
	return c, nil
}

// GetContentByCid returns a post by cid for admin editing (ignores status).
func GetContentByCid(cid int) (Contents, error) {
	fm, body, err := ParseFile(filepath.Join(postsDir, strconv.Itoa(cid)+".md"))
	if err != nil {
		return Contents{}, err
	}
	return ToContents(fm, body, "post", cid), nil
}

// AddComment appends a new comment to a post's front matter.
func AddComment(cidStr, author, mail, url, text string, parent, authorId uint) error {
	cid, err := strconv.Atoi(cidStr)
	if err != nil {
		return fmt.Errorf("invalid cid: %s", cidStr)
	}
	path := filepath.Join(postsDir, strconv.Itoa(cid)+".md")

	mu := getFileMutex(path)
	mu.Lock()
	defer mu.Unlock()

	fm, body, err := ParseFile(path)
	if err != nil {
		return err
	}
	var maxID uint
	for _, c := range fm.Comments {
		if c.ID > maxID {
			maxID = c.ID
		}
	}
	fm.Comments = append(fm.Comments, FMComment{
		ID:      maxID + 1,
		Author:  template.HTMLEscapeString(author),
		Mail:    mail,
		Url:     url,
		Content: template.HTMLEscapeString(text),
		Created: time.Now().Format(time.RFC3339),
		Parent:  parent,
		Status:  "waiting",
	})
	return WriteFile(path, fm, body)
}

// IncrementViews increments the view counter of a post file.
func IncrementViews(cidStr string) error {
	cid, err := strconv.Atoi(cidStr)
	if err != nil {
		return nil
	}
	path := filepath.Join(postsDir, strconv.Itoa(cid)+".md")

	mu := getFileMutex(path)
	mu.Lock()
	defer mu.Unlock()

	fm, body, err := ParseFile(path)
	if err != nil {
		return err
	}
	fm.Views++
	return WriteFile(path, fm, body)
}

// SavePost creates (POST) or updates (PUT) a post file.
func SavePost(method string, cid int, title, slug, text, typ, status, cover, music string, created int64) (int, error) {
	if method == "POST" {
		maxCid := 0
		walkPosts(func(c int, _ FrontMatter, _ string) {
			if c > maxCid {
				maxCid = c
			}
		})
		cid = maxCid + 11
		if slug == "" && typ == "post" {
			slug = strconv.Itoa(cid)
		}
		created = time.Now().Unix()
	}

	path := filepath.Join(postsDir, strconv.Itoa(cid)+".md")
	mu := getFileMutex(path)
	mu.Lock()
	defer mu.Unlock()

	var fm FrontMatter
	if method == "PUT" {
		existing, _, err := ParseFile(path)
		if err == nil {
			fm = existing // preserve views, likes, comments
		}
	}
	fm.Title = title
	fm.Slug = slug
	fm.Created = time.Unix(created, 0).Format(time.RFC3339)
	fm.Cover = cover
	fm.Music = music
	fm.Status = status
	if err := WriteFile(path, fm, text); err != nil {
		return 0, err
	}
	return cid, nil
}

// GetPostsByStatus returns posts with a given status, paginated.
func GetPostsByStatus(status string, limit, offset int) ([]Contents, error) {
	var filtered []Contents
	if err := walkPosts(func(cid int, fm FrontMatter, body string) {
		if fmStatus(fm) == status {
			filtered = append(filtered, ToContents(fm, body, "post", cid))
		}
	}); err != nil {
		return nil, err
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Cid > filtered[j].Cid })
	if offset >= len(filtered) {
		return nil, nil
	}
	return filtered[offset:min(offset+limit+1, len(filtered))], nil
}

// GetAllComments returns comments across all posts filtered by status, paginated.
func GetAllComments(status string, limit, offset int) ([]Comments, error) {
	var all []Comments
	if err := walkPosts(func(cid int, fm FrontMatter, _ string) {
		for _, fc := range fm.Comments {
			if fc.Status == status {
				all = append(all, toComment(fc, uint(cid)))
			}
		}
	}); err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Created > all[j].Created })
	if offset >= len(all) {
		return nil, nil
	}
	return all[offset:min(offset+limit+1, len(all))], nil
}

// ReadConfig reads usr/config.yaml.
func ReadConfig() (Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// GetUser looks up a user by name from config.
func GetUser(name string) (User, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return User{}, err
	}
	if cfg.User.Name != name {
		return User{}, fmt.Errorf("user not found")
	}
	return cfg.User, nil
}
