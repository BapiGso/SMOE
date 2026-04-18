package handler

import (
	"SMOE/moe/mymiddleware"
	"SMOE/moe/store"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
)

func LoginGet(c *echo.Context) error {
	if mymiddleware.IsAdminLoggedIn(c) {
		slog.Info("someone login")
		return c.Render(http.StatusOK, "admin.template", nil)
	}
	return c.Render(http.StatusOK, "login.template", nil)
}

func LoginPost(c *echo.Context) error {
	req := &struct {
		Name     string `form:"user" validate:"required,min=1,max=200"`
		Pwd      string `form:"pwd" validate:"required,min=8,max=200"`
		Illsions string `form:"illsions" `
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	user, err := store.GetUser(req.Name)
	if err != nil {
		return echo.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Pwd)); err == nil {
		if err := mymiddleware.LoginAdmin(c, user.Name); err != nil {
			return err
		}
		return c.Redirect(302, "/admin")
	}
	return echo.ErrUnauthorized
}
