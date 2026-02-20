package store

import (
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FMComment represents a comment stored in YAML front matter.
type FMComment struct {
	ID      uint   `yaml:"id"`
	Author  string `yaml:"author"`
	Mail    string `yaml:"mail"`
	Url     string `yaml:"url,omitempty"`
	Content string `yaml:"content"`
	Created string `yaml:"created"` // RFC3339
	Parent  uint   `yaml:"parent"`
	Status  string `yaml:"status"`
}

// FrontMatter is the YAML header of a .md file.
type FrontMatter struct {
	Title    string      `yaml:"title"`
	Slug     string      `yaml:"slug"`
	Created  string      `yaml:"created"` // RFC3339
	Cover    string      `yaml:"cover,omitempty"`
	Music    string      `yaml:"music,omitempty"`
	Views    uint        `yaml:"views"`
	Likes    uint        `yaml:"likes"`
	Status   string      `yaml:"status,omitempty"` // default "publish"
	Comments []FMComment `yaml:"comments,omitempty"`
}

// ParseFile reads a Markdown file with YAML front matter.
// Convention: file starts with "---\n", YAML block, then "\n---\n", then body.
func ParseFile(path string) (FrontMatter, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FrontMatter{}, "", err
	}
	// Strip the opening "---\n" then split on the closing "\n---"
	content := strings.TrimPrefix(string(data), "---\n")
	yamlStr, body, _ := strings.Cut(content, "\n---")
	body = strings.TrimLeft(body, "\r\n")

	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
		return FrontMatter{}, "", err
	}
	return fm, body, nil
}

// WriteFile writes FrontMatter + body back to a .md file.
// Uses tmp→remove→rename for Windows compatibility (os.Rename fails if target exists).
func WriteFile(path string, fm FrontMatter, body string) error {
	yamlBytes, err := yaml.Marshal(&fm)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(yamlBytes)
	sb.WriteString("---\n")
	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(body)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(sb.String()), 0644); err != nil {
		return err
	}
	os.Remove(path)
	return os.Rename(tmpPath, path)
}

// ToContents converts a FrontMatter + body into a template-ready Contents value.
func ToContents(fm FrontMatter, body, contentType string, cid int) Contents {
	status := fm.Status
	if status == "" {
		status = "publish"
	}
	return Contents{
		Cid:       cid,
		Title:     fm.Title,
		Slug:      fm.Slug,
		Created:   parseRFC3339(fm.Created),
		Text:      body,
		Type:      contentType,
		Status:    status,
		Views:     fm.Views,
		Likes:     fm.Likes,
		CoverList: fm.Cover,
		MusicList: fm.Music,
	}
}

func parseRFC3339(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}
