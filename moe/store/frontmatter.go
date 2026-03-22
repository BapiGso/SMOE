package store

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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
