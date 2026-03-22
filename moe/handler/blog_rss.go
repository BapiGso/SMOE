package handler

import (
	"SMOE/moe/store"
	"encoding/xml"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func RSS(c *echo.Context) error {
	baseURL := c.Scheme() + "://" + c.Request().Host

	posts, _, err := store.GetPostsByCidDesc(20, 0)
	if err != nil {
		return err
	}

	items := make([]rssItem, len(posts))
	for i, p := range posts {
		link := baseURL + "/archives/" + strconv.Itoa(p.Cid)
		items[i] = rssItem{
			Title:       p.Title,
			Link:        link,
			Description: p.MDSub(),
			PubDate:     time.Unix(p.Created, 0).UTC().Format(time.RFC1123Z),
			GUID:        link,
		}
	}

	var lastBuild string
	if len(posts) > 0 {
		lastBuild = time.Unix(posts[0].Created, 0).UTC().Format(time.RFC1123Z)
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         "晓梦的博客",
			Link:          baseURL,
			Description:   "晓梦的博客",
			LastBuildDate: lastBuild,
			Items:         items,
		},
	}

	c.Response().Header().Set(echo.HeaderContentType, "application/rss+xml; charset=utf-8")
	return c.XML(200, feed)
}
