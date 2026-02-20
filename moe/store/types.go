package store

import (
	"SMOE/moe/tools"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"unsafe"
)

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

// MD2HTML markdown转换为html
func (c Contents) MD2HTML() string {
	var buf bytes.Buffer
	_ = tools.GoldMark.Convert(*(*[]byte)(unsafe.Pointer(&c.Text)), &buf)
	return buf.String()
}

// MDSub 截取前70个字符串作为摘要
func (c Contents) MDSub() string {
	length := len([]rune(c.Text))
	if length <= 70 {
		return c.Text
	}
	return string([]rune(c.Text)[:70])
}

// MDCount 计算文章字数
func (c Contents) MDCount() int {
	return utf8.RuneCount(*(*[]byte)(unsafe.Pointer(&c.Text)))
}

func (c Contents) UnixToStr() string {
	monStr := [...]string{"", "一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"}
	mon := int((time.Unix(c.Created, 0)).Month())
	format := (time.Unix(c.Created, 0)).Format("01 02, 2006")
	tmp := strings.Replace(format, format[:2], monStr[mon], 1)
	return tmp
}

func (c Contents) UnixFormat() string {
	return (time.Unix(c.Created, 0)).Format("2006年01月02日")
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

func (c Comments) UnixFormat() string {
	return (time.Unix(c.Created, 0)).Format("2006年01月02日")
}

func (c Comments) MD5Mail() string {
	data := md5.Sum([]byte(c.Mail))
	return fmt.Sprintf("%x", data)
}

func (c Comments) SubText() string {
	runes := []rune(c.Text)
	if len(runes) <= 20 {
		return c.Text
	}
	more := fmt.Sprintf(`...<a class="tooltip" data-tooltip="%v">查看更多</a>`, c.Text)
	return string(runes[:20]) + more
}

// QPU Query Processing Unit 模板数据容器
type QPU struct {
	Contents []Contents
	Comments []Comments
}

var qpuPool = sync.Pool{
	New: func() any {
		return new(QPU)
	},
}

func NewQPU() *QPU {
	return qpuPool.Get().(*QPU)
}

func FreeQPU(q *QPU) {
	q.Contents = q.Contents[:0]
	q.Comments = q.Comments[:0]
	qpuPool.Put(q)
}

// Json 转为json字符串返回以供Alpine JS调用
func (q *QPU) Json() string {
	marshal, err := json.Marshal(q)
	if err != nil {
		return err.Error()
	}
	return string(marshal)
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
