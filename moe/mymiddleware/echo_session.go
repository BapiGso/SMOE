package mymiddleware

import (
	"SMOE/moe/store"
	"crypto/sha256"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	echosession "github.com/labstack/echo-contrib/v5/session"
	"github.com/labstack/echo/v5"
	"github.com/quasoft/memstore"
)

const adminSessionName = "smoe_admin"

var adminSessionStore sessions.Store = memstore.NewMemStore(
	[]byte("01234567890123456789012345678901"),
	[]byte("abcdefghijklmnopqrstuvwxyz123456"),
)

func ConfigureAdminSession(cfg store.Config) {
	authKey := sha256.Sum256([]byte("smoe:session:auth:" + cfg.Name + ":" + cfg.Password))
	encKey := sha256.Sum256([]byte("smoe:session:enc:" + cfg.Name + ":" + cfg.Password))
	store := memstore.NewMemStore(authKey[:], encKey[:])
	store.Options = &sessions.Options{
		Path:     "/admin",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	store.MaxAge(store.Options.MaxAge)
	adminSessionStore = store
}

func Session() echo.MiddlewareFunc {
	return echosession.Middleware(adminSessionStore)
}

func adminSession(c *echo.Context) (*sessions.Session, error) {
	return echosession.Get(adminSessionName, c)
}

func IsAdminLoggedIn(c *echo.Context) bool {
	sess, err := adminSession(c)
	if err != nil {
		return false
	}
	authed, ok := sess.Values["authenticated"].(bool)
	return ok && authed
}

func LoginAdmin(c *echo.Context, user string) error {
	sess, err := adminSession(c)
	if err != nil {
		return err
	}
	sess.Options = &sessions.Options{
		Path:     "/admin",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	sess.Values["authenticated"] = true
	sess.Values["user"] = user
	return sess.Save(c.Request(), c.Response())
}

func RequireAdminSession() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if IsAdminLoggedIn(c) {
				return next(c)
			}
			switch c.Request().Method {
			case http.MethodGet, http.MethodHead:
				return c.Redirect(http.StatusFound, "/admin")
			default:
				return echo.ErrUnauthorized
			}
		}
	}
}
