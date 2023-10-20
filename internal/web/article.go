package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	logger2 "gitee.com/geekbang/basic-go/webook/pkg/logger"
)

var _ handler = (*ArticleHandler)(nil)

type ArticleHandler struct {
	svc service.ArticleService
	l   logger2.LoggerV1
}

func (h *ArticleHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/articles")
	// 在有 list 等路由的时候，无法这样注册

	g.GET("/detail/:id", h.Detail)
	// 理论上来说应该用 GET的，但是我实在不耐烦处理类型转化
	// 直接 POST，JSON 转一了百了。
	g.POST("/list", h.List)

	g.POST("/edit", h.Edit)
	g.POST("/publish", h.Publish)
	g.POST("/withdraw", h.Withdraw)
	pub := g.Group("/pub")
	//pub.GET("/pub", a.PubList)
	pub.GET("/:id", h.PubDetail)

}

func (h *ArticleHandler) Publish(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	c := ctx.MustGet("claims")
	claims, ok := c.(*LUserClaims)
	if !ok {
		// 你可以考虑监控住这里
		//ctx.AbortWithStatus(http.StatusUnauthorized)
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		h.l.Error("未发现用户的 session 信息")
		return
	}

	id, err := h.svc.Publish(ctx, req.toDomain(claims.Uid))
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		// 打日志？
		h.l.Error("发表帖子失败", logger2.Error(err))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg:  "OK",
		Data: id,
	})
}

func (h *ArticleHandler) Edit(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	c := ctx.MustGet("claims")
	claims, ok := c.(*LUserClaims)
	if !ok {
		// 你可以考虑监控住这里
		//ctx.AbortWithStatus(http.StatusUnauthorized)
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		h.l.Error("未发现用户的 session 信息")
		return
	}
	// 检测输入，跳过这一步
	// 调用 svc 的代码
	id, err := h.svc.Save(ctx, req.toDomain(claims.Uid))
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		// 打日志？
		h.l.Error("保存帖子失败", logger2.Error(err))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg:  "OK",
		Data: id,
	})
}

// Withdraw 隐藏
func (h *ArticleHandler) Withdraw(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		h.l.Error("反序列化请求失败", logger2.Error(err))
		return
	}
	usr, ok := ctx.MustGet("user").(*LUserClaims)
	if !ok {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		h.l.Error("获得用户会话信息失败")
		return
	}
	if err := h.svc.Withdraw(ctx, usr.Uid, req.Id); err != nil {
		h.l.Error("设置为尽自己可见失败", logger2.Error(err),
			logger2.Field{Key: "id", Value: req.Id})
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "OK",
	})
}

func (h *ArticleHandler) PubDetail(ctx *gin.Context) {

}

func (h *ArticleHandler) List(ctx *gin.Context) {

}

func (h *ArticleHandler) Detail(ctx *gin.Context) {

}

type ArticleReq struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (req ArticleReq) toDomain(uid int64) domain.Article {
	return domain.Article{
		Id:      req.Id,
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			Id: uid,
		},
	}
}
