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
		Port      string `yaml:"port"`      // HTTP 端口，默认 "80"
		HttpsPort string `yaml:"httpsPort"` // HTTPS 端口（使用内置自签名证书）
		Domain    string `yaml:"domain"`    // 域名，非空时启用 Let's Encrypt
	} `yaml:"server"`
}
