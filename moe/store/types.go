package store

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

// QPU Query Processing Unit 模板数据容器
type QPU struct {
	Contents      []Contents
	Comments      []Comments
	CommentGroups [][]Comments
}

type User struct {
	Name       string `yaml:"name"`
	Password   string `yaml:"password"`
	Mail       string `yaml:"mail"`
	ScreenName string `yaml:"screenName"`
}

type Config struct {
	User    User `yaml:"user"`
	Bangumi struct {
		UserID string `yaml:"userId"`
		AppID  string `yaml:"appId"`
	} `yaml:"bangumi"`
	Server struct {
		Port string `yaml:"port"` // HTTP 端口，默认 "80"
	} `yaml:"server"`
	Mail struct {
		ResendAPI string `yaml:"resendAPI"`
		To        string `yaml:"to"`
		CC        string `yaml:"cc"`
	} `yaml:"mail"`
}
