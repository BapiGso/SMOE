package store

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"
)

// --- Contents methods ---

// MD2HTML markdown转换为html
func (c Contents) MD2HTML() string {
	var buf bytes.Buffer
	_ = goldMark.Convert(*(*[]byte)(unsafe.Pointer(&c.Text)), &buf)
	return buf.String()
}

// MDSub 截取前70个字符作为摘要（Markdown→HTML→纯文本）
func (c Contents) MDSub() string {
	html := c.MD2HTML()
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	plain := strings.Join(strings.Fields(b.String()), " ")
	runes := []rune(plain)
	if len(runes) <= 70 {
		return plain
	}
	return string(runes[:70])
}

// MDCount 计算文章字数
func (c Contents) MDCount() int {
	return utf8.RuneCount(*(*[]byte)(unsafe.Pointer(&c.Text)))
}

func (c Contents) UnixToStr() string {
	monStr := [...]string{"", "一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"}
	mon := int((time.Unix(c.Created, 0)).Month())
	format := (time.Unix(c.Created, 0)).Format("01 02, 2006")
	return strings.Replace(format, format[:2], monStr[mon], 1)
}

func (c Contents) UnixFormat() string {
	return (time.Unix(c.Created, 0)).Format("2006年01月02日")
}

// --- Comments methods ---

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

// --- QPU methods ---

// GroupComments 按线程分组评论：每个父评论及其子评论组成一组。
func GroupComments(data []Comments) [][]Comments {
	var groups [][]Comments
	parentMap := make(map[uint]int)
	for _, v := range data {
		if v.Parent == 0 {
			groups = append(groups, []Comments{v})
			parentMap[v.Coid] = len(groups) - 1
		} else {
			index := parentMap[v.Parent]
			groups[index] = append(groups[index], v)
			parentMap[v.Coid] = index
		}
	}
	return groups
}

// Json 转为json字符串以供Alpine JS调用
func (q *QPU) Json() string {
	marshal, err := json.Marshal(q)
	if err != nil {
		return err.Error()
	}
	return string(marshal)
}
