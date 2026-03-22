package store

// --- Content types ---

type Contents struct {
	Cid       int    `json:"Cid"`
	Title     string `json:"Title"`
	Slug      string `json:"Slug"`
	Created   int64  `json:"Created"`
	Text      string `json:"Text"`
	Type      string `json:"Type"`
	Status    string `json:"Status"`
	Views     uint   `json:"Views"`
	Likes     uint   `json:"Likes"`
	CoverList string `json:"CoverList"`
	AutoCover string `json:"AutoCover"`
	MusicList string `json:"MusicList"`
}

type Comments struct {
	Coid     uint    `json:"Coid"`
	Cid      uint    `json:"Cid"`
	Created  int64   `json:"Created"`
	Author   string  `json:"Author"`
	AuthorId uint    `json:"AuthorId"`
	Mail     string  `json:"Mail"`
	Url      *string `json:"Url"`
	Ip       string  `json:"Ip"`
	Agent    string  `json:"Agent"`
	Text     string  `json:"Text"`
	Status   string  `json:"Status"`
	Parent   uint    `json:"Parent"`
}

type CommentNotification struct {
	PostTitle string
	PostCID   uint
	Comment   Comments
	Parent    *Comments
}

// --- Front matter types ---

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
	Cover    string      `yaml:"cover,omitempty"`
	Music    string      `yaml:"music,omitempty"`
	Views    uint        `yaml:"views"`
	Likes    uint        `yaml:"likes"`
	Status   string      `yaml:"status,omitempty"` // default "publish"
	Comments []FMComment `yaml:"comments,omitempty"`
}

// --- Config types ---

type Config struct {
	Name          string `yaml:"name"`
	Password      string `yaml:"password"`
	Mail          string `yaml:"mail"`
	BangumiUserID string `yaml:"bangumiUserID"`
	BangumiAppID  string `yaml:"bangumiAppID"`
	Port          string `yaml:"port"` // HTTP 端口，默认 "95"
	ResendAPI     string `yaml:"resendAPI"`
	MailTo        string `yaml:"mailTo"`
	MailCC        string `yaml:"mailCC"`
}

// --- QPU (Query Processing Unit) 模板数据容器 ---

type QPU struct {
	Contents      []Contents
	Comments      []Comments
	CommentGroups [][]Comments
}

// ToContents converts a FrontMatter + body into a template-ready Contents value.
// cid is the Unix timestamp derived from the filename; title comes from the filename.
func ToContents(fm FrontMatter, title, body, contentType string, cid int) Contents {
	status := fm.Status
	if status == "" {
		status = "publish"
	}
	return Contents{
		Cid:       cid,
		Title:     title,
		Created:   int64(cid),
		Text:      body,
		Type:      contentType,
		Status:    status,
		Views:     fm.Views,
		Likes:     fm.Likes,
		CoverList: fm.Cover,
		MusicList: fm.Music,
	}
}
