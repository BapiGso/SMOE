package mymiddleware

// InsightLog todo 不要依赖database
//func InsightLog(next echo.HandlerFunc) echo.HandlerFunc {
//	return func(c *echo.Context) error {
//		c.Response().After(func() {
//			_, err := database.DB.Exec(`INSERT INTO smoe_insights (ua, url, path, ip, referer, time) VALUES (?, ?, ?, ?, ?, ?)`,
//				c.Request().UserAgent(), c.Request().RequestURI, c.Request().URL.Path, c.RealIP(), c.Request().Referer(), time.Now().Unix())
//			if err != nil {
//				//todo database is locked (5) (SQLITE_BUSY)
//				slog.Error(err.Error())
//			}
//		})
//		return next(c)
//	}
//}
