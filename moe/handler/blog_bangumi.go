package handler

import (
	"SMOE/moe/database"
	"encoding/json"
	"github.com/labstack/echo/v5"
	"io"
	"net/http"
	"sort"
	"time"
)

const (
	// SubjectType
	// Book 书籍
	book = 1
	// Animation 动画
	animation = 2
	// Music 音乐
	music = 3
	// Game 游戏
	game = 4
	// ThreeD 三次元
	threeD = 5

	// CollectionType
	// WantToWatch 想看
	wantToWatch = 1
	// Watched 看过
	watched = 2
	// Watching 在看
	watching = 3
	// OnHold 搁置
	onHold = 4
	// Dropped 抛弃
	dropped = 5
)

type OldBangumi []struct {
	Type     int    `json:"type"`
	Name     string `json:"name"`
	NameCn   string `json:"name_cn"`
	Collects []struct {
		Status struct {
			ID   int    `json:"id"`
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"status"`
		Count int `json:"count"`
		List  []struct {
			SubjectID int `json:"subject_id"`
			Subject   struct {
				ID         int    `json:"id"`
				URL        string `json:"url"`
				Type       int    `json:"type"`
				Name       string `json:"name"`
				NameCn     string `json:"name_cn"`
				Summary    string `json:"summary"`
				AirDate    string `json:"air_date"`
				AirWeekday int    `json:"air_weekday"`
				Images     struct {
					Large  string `json:"large"`
					Common string `json:"common"`
					Medium string `json:"medium"`
					Small  string `json:"small"`
					Grid   string `json:"grid"`
				} `json:"images"`
			} `json:"subject"`
		} `json:"list"`
	} `json:"collects"`
}

type NewBangumi struct {
	Data []struct {
		UpdatedAt time.Time     `json:"updated_at"`
		Comment   interface{}   `json:"comment"`
		Tags      []interface{} `json:"tags"`
		Subject   struct {
			Date   string `json:"date"`
			Images struct {
				Small  string `json:"small"`
				Grid   string `json:"grid"`
				Large  string `json:"large"`
				Medium string `json:"medium"`
				Common string `json:"common"`
			} `json:"images"`
			Name         string `json:"name"`
			NameCn       string `json:"name_cn"`
			ShortSummary string `json:"short_summary"`
			Tags         []struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			} `json:"tags"`
			Score           float64 `json:"score"`
			Type            int     `json:"type"`
			ID              int     `json:"id"`
			Eps             int     `json:"eps"`
			Volumes         int     `json:"volumes"`
			CollectionTotal int     `json:"collection_total"`
			Rank            int     `json:"rank"`
		} `json:"subject"`
		SubjectID   int  `json:"subject_id"`
		VolStatus   int  `json:"vol_status"`
		EpStatus    int  `json:"ep_status"`
		SubjectType int  `json:"subject_type"`
		Type        int  `json:"type"`
		Rate        int  `json:"rate"`
		Private     bool `json:"private"`
	} `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func curlBGM(url string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Set("User-Agent", "trim21/bangumi-episode-ics (https://github.com/Trim21/bangumi-episode-calendar)")
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	m := NewBangumi{}
	if err := json.Unmarshal(body, &m); err != nil {
		return err
	}
	sort.Slice(m.Data, func(i, j int) bool {
		return m.Data[i].Subject.ID < m.Data[j].Subject.ID
	})
	bgm = bgmCache{nil, m, time.Now().Unix()}
	return err
}

type bgmCache struct {
	OldBangumi OldBangumi
	NewBangumi NewBangumi
	TTL        int64
}

var bgm = bgmCache{}

// Bangumi todo https://freefrontend.com/css-cards/
func Bangumi(c *echo.Context) error {
	qpu := new(database.QPU)
	if err := database.DB.Get(&qpu.Options, `SELECT * FROM smoe_options WHERE name = 'Goplugin:BangumiList'`); err != nil {
		return err
	}
	m := struct {
		UserID string
		AppID  string
	}{}
	if err := json.Unmarshal([]byte(qpu.Options.Value), &m); err != nil {
		return err
	}
	//oldAPI := "https://api.bgm.tv/user/" + m.UserID + "/collections/anime?app_id=" + m.AppID + "&max_results=99"
	newAPI := "https://api.bgm.tv/v0/users/" + m.UserID + "/collections?subject_type=2&limit=100&offset=0"
	//每七天更新一下
	if time.Now().Unix()-bgm.TTL > 604800 {
		if err := curlBGM(newAPI); err != nil {
			return err
		}
	}
	return c.Render(200, "page-bangumi.template", bgm.NewBangumi)
}
