// Package blogdata 管理程序自己写的数据：usr/data/ 下的浏览量、评论和番剧。
// Hugo 把这个目录当作数据源读取（模板里的 hugo.Data），用户不应手动修改。
package blogdata

import (
	"os"
	"path/filepath"
)

// Comment 是 data/comments/<slug>.yaml 里的一条评论。
type Comment struct {
	ID      uint   `yaml:"id"`
	Author  string `yaml:"author"`
	Mail    string `yaml:"mail"`
	URL     string `yaml:"url,omitempty"`
	Content string `yaml:"content"`
	Created string `yaml:"created"` // RFC3339
	Parent  uint   `yaml:"parent"`
	Status  string `yaml:"status"` // waiting / approved / spam
}

// Stat 是 data/stats.json 里一篇文章的计数。
type Stat struct {
	Views uint `json:"views"`
	Likes uint `json:"likes"`
}

// WriteFileAtomic 先写临时文件再改名，读者（Hugo 构建）不会读到写了一半的文件。
// 临时文件以 .tmp 结尾，引擎的监听器会忽略它。
func WriteFileAtomic(name string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(b); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, name); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
