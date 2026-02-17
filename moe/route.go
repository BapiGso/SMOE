package moe

import (
	"SMOE/moe/handler"
	"SMOE/moe/mymiddleware"
	"net/http"
	"text/template"

	"github.com/labstack/echo/v5/middleware"
)

func (s *Smoe) LoadMiddlewareRoutes() {
	s.e.Validator = &mymiddleware.Validator{}
	s.e.Renderer = &mymiddleware.TemplateRender{
		Template: template.Must(
			template.ParseFS(
				s.themeFS,
				"blog/*.template",
				"blog/css/*.css",
				"new-admin/*.template",
			),
		).Funcs(template.FuncMap{}),
	}
	//Secure防XSS，HSTS防中间人攻击 todo 防盗链
	s.e.Pre(middleware.SecureWithConfig(middleware.SecureConfig{
		HSTSMaxAge:            31536000,
		HSTSPreloadEnabled:    true,
		HSTSExcludeSubdomains: true,
	}))
	//cors防盗链
	//s.e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
	//	AllowOrigins: []string{"http://localhost:8080"}, // 允许的源地址
	//	AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	//}))
	// 中间件：禁用跨站请求伪造（CSRF）
	//s.e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
	//	TokenLookup: "header:X-CSRF-Token", // 从请求头中获取CSRF令牌
	//}))

	//s.e.Logger.SetLevel(log.INFO)
	s.e.Use(mymiddleware.Slog())
	//s.e.Use(souinecho.NewMiddleware(souinemiddleware.BaseConfiguration{}).Process)

	//s.e.Use(middleware.Logger())
	s.e.Use(middleware.Recover())

	s.e.Pre(mymiddleware.Brotli())

	//http重定向https
	//s.e.Pre(middleware.HTTPSRedirect())

	//使用jwt控制后台访问
	s.e.Use(mymiddleware.JWT())

	s.e.StaticFS("/assets", s.themeFS)

	s.e.HTTPErrorHandler = handler.FrontErr //自定义错误页面
	front := s.e.Group("")
	back := s.e.Group("/admin")

	// 前台页面路由
	//front.Use(mymiddleware.InsightLog)
	//301跳转去除尾部斜杠
	front.Use(middleware.RemoveTrailingSlashWithConfig(middleware.RemoveTrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	front.GET("/", handler.Index)                                      // 首页路由
	front.GET("/page/:num", handler.Index)                             // 分页路由，显示指定页数的文章列表
	front.GET("/archives/:cid", handler.Post)                          // 根据分类ID显示该分类下的文章列表
	front.POST("/archives/:cid/comment", handler.SubmitArticleComment) // 管理评论提交
	front.GET("/:page", handler.Page)                                  // 独立页面，注册在特殊独立页面前
	front.GET("/archives", handler.Archives)                           // 归档页面路由，显示所有文章的归档分类
	front.GET("/bangumi", handler.Bangumi)                             // 显示番剧相关信息的页面路由
	front.Static("/usr/uploads", "usr/uploads")                        // 用户上传的文件，最后注册
	// 后台管理
	// 后台管理的路由组
	back.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(3))) //每秒限制3次请求
	// 后台管理页面路由
	back.GET("", handler.LoginGet)   // 后台管理登录页面路由
	back.POST("", handler.LoginPost) // 后台管理登录处理路由
	back.Any("/write/:cid", handler.Write)
	back.Any("/manage/:type", handler.Manage)
	back.GET("/insight", handler.Insight)
	back.GET("/setting", handler.Setting)
	// 文件上传路由
	back.POST("/upload", handler.Upload) // 处理图片上传请求的路由
}
