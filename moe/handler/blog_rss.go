package handler

import (
	"SMOE/moe/store"
	"strconv"
	"time"

	"github.com/gorilla/feeds"
	"github.com/labstack/echo/v5"
)

// RSS 以 gorilla/feeds 同时生成 RSS 2.0。
// 如需 Atom / JSON Feed 只需换一个方法即可。
func RSS(c *echo.Context) error {
	baseURL := c.Scheme() + "://" + c.Request().Host

	posts, _, err := store.GetPostsByCidDesc(20, 0)
	if err != nil {
		return err
	}

	feed := &feeds.Feed{
		Title:       "晓梦的博客",
		Link:        &feeds.Link{Href: baseURL},
		Description: "晓梦的博客",
	}
	if len(posts) > 0 {
		feed.Created = time.Unix(posts[0].Created, 0)
	}

	feed.Items = make([]*feeds.Item, len(posts))
	for i, p := range posts {
		link := baseURL + "/archives/" + strconv.Itoa(p.Cid)
		feed.Items[i] = &feeds.Item{
			Id:          link,
			Title:       p.Title,
			Link:        &feeds.Link{Href: link},
			Description: p.MDSub(),
			Created:     time.Unix(p.Created, 0),
		}
	}

	c.Response().Header().Set(echo.HeaderContentType, "application/rss+xml; charset=utf-8")
	rss, err := feed.ToRss()
	if err != nil {
		return err
	}
	return c.String(200, rss)
}
