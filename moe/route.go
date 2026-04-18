package moe

import (
	"SMOE/moe/handler"
	"SMOE/moe/mymiddleware"
	"html/template"
	"net/http"

	"github.com/labstack/echo/v5/middleware"
)

func (s *Smoe) LoadMiddlewareRoutes() {
	s.e.Validator = &mymiddleware.Validator{}
	s.e.Renderer = &mymiddleware.TemplateRender{
		Template: template.Must(
			template.ParseFS(
				s.themeFS,
				"blog/*.template",
				"admin/*.template",
			),
		),
	}
	// WebDAV 文件挂载：/webdav/* → usr/，Basic Auth
	s.e.Pre(mymiddleware.WebDAV())

	//Secure防XSS，HSTS防中间人攻击 todo 防盗链
	s.e.Pre(middleware.SecureWithConfig(middleware.SecureConfig{
		HSTSMaxAge:            31536000,
		HSTSPreloadEnabled:    true,
		HSTSExcludeSubdomains: true,
	}))

	s.e.Use(mymiddleware.Slog())
	s.e.Use(middleware.Recover())

	s.e.StaticFS("/assets", s.themeFS)

	s.e.HTTPErrorHandler = handler.FrontErr //自定义错误页面
	front := s.e.Group("")
	back := s.e.Group("/admin")

	// 前台页面路由
	//301跳转去除尾部斜杠
	front.Use(middleware.RemoveTrailingSlashWithConfig(middleware.RemoveTrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	front.GET("/", handler.IndexGorm)                                                                                                                                // 首页路由
	front.GET("/page/:num", handler.IndexGorm)                                                                                                                       // 分页路由
	front.GET("/archives/:cid", handler.Post)                                                                                                                        // 根据分类ID显示该分类下的文章列表
	front.POST("/archives/:cid/comment", handler.SubmitArticleComment, mymiddleware.SameOriginOnly(), mymiddleware.CommentRateLimit(), mymiddleware.CommentNotify()) // 管理评论提交
	front.POST("/archives/:cid/like", handler.LikePost, mymiddleware.SameOriginOnly(), mymiddleware.LikeRateLimit())                                                 // 文章点赞
	front.GET("/:page", handler.Page)                                                                                                                                // 独立页面，注册在特殊独立页面前
	front.GET("/feed", handler.RSS)                                                                                                                                  // RSS 订阅
	front.GET("/archives", handler.Archives)                                                                                                                         // 归档页面路由，显示所有文章的归档分类
	front.GET("/bangumi", handler.Bangumi)                                                                                                                           // 显示番剧相关信息的页面路由
	front.Static("/usr/uploads", "usr/uploads")                                                                                                                      // 用户上传的文件，最后注册

	// 后台管理的路由组
	back.Use(mymiddleware.Session())
	back.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(3))) //每秒限制3次请求
	backSecure := back.Group("", mymiddleware.RequireAdminSession())
	// 后台管理页面路由
	back.GET("", handler.LoginGet)                            // 后台管理登录页面路由
	back.POST("", handler.LoginPost, mymiddleware.LoginRateLimit()) // 后台管理登录处理路由（带防爆破限流）
	backSecure.Any("/write/:cid", handler.Write)
	backSecure.Any("/manage/:type", handler.Manage)
	backSecure.GET("/insight", handler.Insight)
	backSecure.GET("/setting", handler.Setting)
	// 文件上传路由
	backSecure.POST("/upload", handler.Upload) // 处理图片上传请求的路由
}
