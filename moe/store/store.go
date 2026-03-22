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
	coverDir   = "usr/uploads/background"
	configFile = "usr/config.yaml"
)

// --- concurrency ---

var fileMutexes sync.Map

func parseRFC3339(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

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

// cidFromName extracts the cid from a post filename like "1586584980-标题.md" or "1586584980.md".
func cidFromName(name string) (int, bool) {
	if i := strings.IndexByte(name, '-'); i > 0 {
		cid, err := strconv.Atoi(name[:i])
		return cid, err == nil
	}
	cid, err := strconv.Atoi(name)
	return cid, err == nil
}

// titleFromName extracts the title part from a filename like "1586584980-标题".
func titleFromName(name string) string {
	if i := strings.IndexByte(name, '-'); i > 0 {
		return name[i+1:]
	}
	return ""
}

// postPath finds the actual file path for a given cid in postsDir.
// Supports both "{cid}.md" and "{cid}-{title}.md" naming.
// Also returns the title extracted from the filename.
func postPath(cid int) (path string, title string, err error) {
	prefix := strconv.Itoa(cid)
	entries, err := os.ReadDir(postsDir)
	if err != nil {
		return "", "", err
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		if name == prefix {
			return filepath.Join(postsDir, e.Name()), "", nil
		}
		if strings.HasPrefix(name, prefix+"-") {
			return filepath.Join(postsDir, e.Name()), titleFromName(name), nil
		}
	}
	return "", "", fmt.Errorf("post %d not found", cid)
}

// postFileName builds a filename like "{cid}-{title}.md".
func postFileName(cid int, title string) string {
	safe := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_").Replace(title)
	return strconv.Itoa(cid) + "-" + safe + ".md"
}

// walkPosts calls fn for every valid post in usr/posts/*.md.
func walkPosts(fn func(cid int, title string, fm FrontMatter, body string)) error {
	entries, err := os.ReadDir(postsDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		cid, valid := cidFromName(name)
		if !valid {
			continue
		}
		fm, body, err := ParseFile(filepath.Join(postsDir, e.Name()))
		if err != nil {
			continue
		}
		fn(cid, titleFromName(name), fm, body)
	}
	return nil
}

func listAutoCovers() ([]string, error) {
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	covers := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".avif") {
			continue
		}
		covers = append(covers, "/"+e.Name())
	}
	sort.Strings(covers)
	return covers, nil
}

func assignAutoCovers(posts []Contents) error {
	covers, err := listAutoCovers()
	if err != nil || len(covers) == 0 {
		return err
	}
	for i := range posts {
		posts[i].AutoCover = "/usr/uploads/background" + covers[i%len(covers)]
	}
	return nil
}

// toComment converts a single FMComment to a Comments value.
func toComment(fc FMComment, cid uint) Comments {
	var url *string
	if fc.Url != "" {
		url = new(fc.Url)
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
	if err := walkPosts(func(cid int, title string, fm FrontMatter, body string) {
		if fmStatus(fm) == "publish" {
			all = append(all, ToContents(fm, title, body, "post", cid))
		}
	}); err != nil {
		return nil, false, err
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Cid > all[j].Cid })
	if err := assignAutoCovers(all); err != nil {
		return nil, false, err
	}

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
	if err := walkPosts(func(cid int, title string, fm FrontMatter, body string) {
		if fmStatus(fm) == "publish" {
			result = append(result, ToContents(fm, title, body, "post", cid))
		}
	}); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Cid > result[j].Cid })
	if err := assignAutoCovers(result); err != nil {
		return nil, err
	}
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
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		fm, body, err := ParseFile(filepath.Join(pagesDir, e.Name()))
		if err != nil {
			continue
		}
		slug, title, _ := strings.Cut(name, "-")
		c := ToContents(fm, title, body, "page", 0)
		c.Slug = slug
		pages = append(pages, c)
	}
	return pages, nil
}

func autoCoverForCid(cid int) (string, error) {
	posts, err := GetAllPublishedPosts()
	if err != nil {
		return "", err
	}
	for _, post := range posts {
		if post.Cid == cid {
			return post.AutoCover, nil
		}
	}
	return "", nil
}

// GetPostByCid returns a published post and its approved comments.
func GetPostByCid(cid int) (Contents, []Comments, error) {
	path, title, err := postPath(cid)
	if err != nil {
		return Contents{}, nil, err
	}
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
	content := ToContents(fm, title, body, "post", cid)
	cover, err := autoCoverForCid(cid)
	if err != nil {
		return Contents{}, nil, err
	}
	content.AutoCover = cover
	return content, approved, nil
}

// pagePath finds the actual file path and title for a given slug in pagesDir.
func pagePath(slug string) (path string, title string, err error) {
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		return "", "", err
	}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok {
			continue
		}
		if name == slug {
			return filepath.Join(pagesDir, e.Name()), "", nil
		}
		if strings.HasPrefix(name, slug+"-") {
			return filepath.Join(pagesDir, e.Name()), name[len(slug)+1:], nil
		}
	}
	return "", "", fmt.Errorf("page %s not found", slug)
}

// GetPageBySlug returns a page by its slug.
func GetPageBySlug(slug string) (Contents, error) {
	path, title, err := pagePath(slug)
	if err != nil {
		return Contents{}, err
	}
	fm, body, err := ParseFile(path)
	if err != nil {
		return Contents{}, err
	}
	c := ToContents(fm, title, body, "page", 0)
	c.Slug = slug
	return c, nil
}

// GetContentByCid returns a post by cid for admin editing (ignores status).
func GetContentByCid(cid int) (Contents, error) {
	path, title, err := postPath(cid)
	if err != nil {
		return Contents{}, err
	}
	fm, body, err := ParseFile(path)
	if err != nil {
		return Contents{}, err
	}
	return ToContents(fm, title, body, "post", cid), nil
}

// AddComment appends a new comment to a post's front matter.
func AddComment(cidStr, author, mail, url, text string, parent, authorId uint) (CommentNotification, error) {
	cid, err := strconv.Atoi(cidStr)
	if err != nil {
		return CommentNotification{}, fmt.Errorf("invalid cid: %s", cidStr)
	}
	path, title, err := postPath(cid)
	if err != nil {
		return CommentNotification{}, err
	}

	mu := getFileMutex(path)
	mu.Lock()
	defer mu.Unlock()

	fm, body, err := ParseFile(path)
	if err != nil {
		return CommentNotification{}, err
	}
	var maxID uint
	var parentComment *Comments
	for _, c := range fm.Comments {
		if c.ID > maxID {
			maxID = c.ID
		}
		if c.ID == parent {
			parentComment = new(toComment(c, uint(cid)))
		}
	}
	if parent != 0 && parentComment == nil {
		return CommentNotification{}, fmt.Errorf("parent comment %d not found", parent)
	}

	created := time.Now().Format(time.RFC3339)
	newComment := FMComment{
		ID:      maxID + 1,
		Author:  template.HTMLEscapeString(author),
		Mail:    mail,
		Url:     url,
		Content: template.HTMLEscapeString(text),
		Created: created,
		Parent:  parent,
		Status:  "waiting",
	}
	fm.Comments = append(fm.Comments, newComment)
	if err := WriteFile(path, fm, body); err != nil {
		return CommentNotification{}, err
	}

	comment := toComment(newComment, uint(cid))
	comment.AuthorId = authorId
	return CommentNotification{
		PostTitle: title,
		PostCID:   uint(cid),
		Comment:   comment,
		Parent:    parentComment,
	}, nil
}

// IncrementViews increments the view counter of a post file.
func IncrementViews(cidStr string) error {
	cid, err := strconv.Atoi(cidStr)
	if err != nil {
		return nil
	}
	path, _, err := postPath(cid)
	if err != nil {
		return err
	}

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

// IncrementLikes increments the like counter of a post file.
func IncrementLikes(cidStr string) error {
	cid, err := strconv.Atoi(cidStr)
	if err != nil {
		return nil
	}
	path, _, err := postPath(cid)
	if err != nil {
		return err
	}

	mu := getFileMutex(path)
	mu.Lock()
	defer mu.Unlock()

	fm, body, err := ParseFile(path)
	if err != nil {
		return err
	}
	fm.Likes++
	return WriteFile(path, fm, body)
}

// SavePost creates (POST) or updates (PUT) a post file.
func SavePost(method string, cid int, title, text, status, cover, music string) (int, error) {
	if method == "POST" {
		cid = int(time.Now().Unix())
	}

	newPath := filepath.Join(postsDir, postFileName(cid, title))

	var fm FrontMatter
	if method == "PUT" {
		oldPath, _, err := postPath(cid)
		if err == nil {
			existing, _, parseErr := ParseFile(oldPath)
			if parseErr == nil {
				fm = existing // preserve views, likes, comments
			}
			// remove old file if name changed
			if oldPath != newPath {
				os.Remove(oldPath)
			}
		}
	}

	mu := getFileMutex(newPath)
	mu.Lock()
	defer mu.Unlock()

	fm.Cover = cover
	fm.Music = music
	fm.Status = status
	if err := WriteFile(newPath, fm, text); err != nil {
		return 0, err
	}
	return cid, nil
}

// GetPostsByStatus returns posts with a given status, paginated.
func GetPostsByStatus(status string, limit, offset int) ([]Contents, error) {
	var filtered []Contents
	if err := walkPosts(func(cid int, title string, fm FrontMatter, body string) {
		if fmStatus(fm) == status {
			filtered = append(filtered, ToContents(fm, title, body, "post", cid))
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
	if err := walkPosts(func(cid int, _ string, fm FrontMatter, _ string) {
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
func GetUser(name string) (Config, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return Config{}, err
	}
	if cfg.Name != name {
		return Config{}, fmt.Errorf("user not found")
	}
	return cfg, nil
}
