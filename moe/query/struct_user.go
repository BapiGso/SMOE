package query

// 内存对齐 https://geektutu.com/post/hpg-struct-alignment.html
type User struct {
	Uid        string `db:"uid"`
	Name       string `db:"name"`
	Password   string `db:"password"`
	Mail       string `db:"mail"`
	Url        string `db:"url"`
	ScreenName string `db:"screenName"`
	Created    int64  `db:"created"`
	Activated  int64  `db:"activated"`
	Logged     int64  `db:"logged"`
	Group      string `db:"group"`
	AuthCode   string `db:"authCode"`
}
