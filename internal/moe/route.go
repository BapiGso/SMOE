package moe

import (
	"net/http"

	"github.com/BapiGso/SMOE/internal/moe/handler"
	"github.com/BapiGso/SMOE/internal/moe/mymiddleware"

	"github.com/labstack/echo/v5/middleware"
)

func (s *Smoe) LoadMiddlewareRoutes() {
	s.e.Validator = &mymiddleware.Validator{}
	// WebDAV 文件挂载：/webdav/* → usr/，Basic Auth。写文章、传图片、改配置都走这里
	s.e.Pre(mymiddleware.WebDAV())

	//Secure防XSS，HSTS防中间人攻击 todo 防盗链
	s.e.Pre(middleware.SecureWithConfig(middleware.SecureConfig{
		HSTSMaxAge:            31536000,
		HSTSPreloadEnabled:    true,
		HSTSExcludeSubdomains: true,
	}))

	s.e.Use(mymiddleware.Slog())
	s.e.Use(middleware.Recover())

	s.e.HTTPErrorHandler = handler.FrontErr(s.site) //自定义错误页面
	front := s.e.Group("")

	//301跳转去除尾部斜杠
	front.Use(middleware.RemoveTrailingSlashWithConfig(middleware.RemoveTrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	front.GET("/*", handler.Front(s.site, s.data))                                                                                                                                          // 首页、分页、文章、独立页面、时间线、追番、RSS、css/js 都来自 Hugo 构建产物
	front.POST("/archives/:cid/comment", handler.SubmitArticleComment(s.site, s.data, s.mod), mymiddleware.SameOriginOnly(), mymiddleware.CommentRateLimit(), mymiddleware.CommentNotify()) // 评论提交，待审核
	front.POST("/archives/:cid/like", handler.LikePost(s.site, s.data), mymiddleware.SameOriginOnly(), mymiddleware.LikeRateLimit())                                                        // 文章点赞
	front.Match([]string{http.MethodGet, http.MethodPost}, "/comment/moderate", s.mod.Handler(s.site, s.data))                                                                              // 邮件里的审核链接，签名即凭证
	front.Static("/usr/uploads", "usr/uploads")                                                                                                                                             // 用户上传的文件
}
