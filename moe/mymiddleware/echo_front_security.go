package mymiddleware

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func SameOriginOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			expected := c.Scheme() + "://" + c.Request().Host

			origin := c.Request().Header.Get("Origin")
			if origin != "" {
				if origin != expected {
					return echo.ErrForbidden
				}
				return next(c)
			}

			referer := c.Request().Referer()
			if referer == "" {
				return echo.ErrForbidden
			}
			u, err := url.Parse(referer)
			if err != nil {
				return echo.ErrForbidden
			}
			if u.Scheme+"://"+u.Host != expected {
				return echo.ErrForbidden
			}
			return next(c)
		}
	}
}

func CommentRateLimit() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      1.0 / 60.0,
			Burst:     1,
			ExpiresIn: 2 * time.Minute,
		}),
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			return "comment:" + c.RealIP(), nil
		},
		DenyHandler: func(c *echo.Context, _ string, _ error) error {
			return echo.NewHTTPError(http.StatusTooManyRequests, "评论太频繁，请稍后再试")
		},
	})
}

func LikeRateLimit() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      1.0 / float64((6 * time.Hour).Seconds()),
			Burst:     1,
			ExpiresIn: 6 * time.Hour,
		}),
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			return fmt.Sprintf("like:%s:%s", c.RealIP(), c.Param("cid")), nil
		},
		DenyHandler: func(c *echo.Context, _ string, _ error) error {
			return c.JSON(http.StatusOK, map[string]any{"liked": false})
		},
	})
}

// LoginRateLimit 管理员登录限流：单个 IP 每 10 分钟最多 5 次尝试。
// 失败返回 429，避免爆破。
func LoginRateLimit() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      5.0 / float64((10 * time.Minute).Seconds()),
			Burst:     5,
			ExpiresIn: 30 * time.Minute,
		}),
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			return "login:" + c.RealIP(), nil
		},
		DenyHandler: func(c *echo.Context, _ string, _ error) error {
			return echo.NewHTTPError(http.StatusTooManyRequests, "登录尝试过多，请稍后再试")
		},
	})
}
