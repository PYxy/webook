package web

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/demdxx/gocast/v2"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/service"
	"gitee.com/geekbang/basic-go/webook/internal/web/jwt"
	"gitee.com/geekbang/basic-go/webook/pkg/ginx/logger3"
	logger2 "gitee.com/geekbang/basic-go/webook/pkg/logger"
)

var _ handler = (*ArticleHandler)(nil)

type ArticleHandler struct {
	svc     service.ArticleService
	l       logger2.LoggerV1
	intrSvc service.InteractiveService
	biz     string
}

func NewArticleHandler(svc service.ArticleService,
	intrSvc service.InteractiveService,
	l logger2.LoggerV1) *ArticleHandler {
	return &ArticleHandler{
		svc:     svc,
		intrSvc: intrSvc,
		l:       l,
		biz:     "article",
	}
}

func (h *ArticleHandler) RegisterPublicRoutes(server *gin.Engine) {
	g := server.Group("/articles")
	g.GET("/topN/:key/:top", h.GetTop)
}

func (h *ArticleHandler) RegisterPrivateRoutes(server *gin.Engine) {
	g := server.Group("/articles")
	// 在有 list 等路由的时候，无法这样注册

	//g.GET("/detail/:id", h.Detail)
	// 理论上来说应该用 GET的，但是我实在不耐烦处理类型转化
	// 直接 POST，JSON 转一了百了。
	g.POST("/list",
		logger3.WrapReq[ListReq](h.List))
	g.GET("/detail/:id", logger3.WrapToken[jwt.UserClaims](h.Detail))
	g.POST("/edit", h.Edit)
	g.POST("/publish", h.Publish)
	g.POST("/withdraw", h.Withdraw)
	pub := g.Group("/pub")
	//pub.GET("/pub", a.PubList)
	pub.GET("/:id", h.PubDetail, func(ctx *gin.Context) {
		// 增加阅读计数。
		//go func() {
		//	// 开一个 goroutine，异步去执行
		//	er := a.intrSvc.IncrReadCnt(ctx, a.biz, art.Id)
		//	if er != nil {
		//		a.l.Error("增加阅读计数失败",
		//			logger.Int64("aid", art.Id),
		//			logger.Error(err))
		//	}
		//}()
	})
	pub.POST("/like", logger3.WrapReq[LikeReq](h.Like))
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
	usr, ok := ctx.MustGet("claims").(*LUserClaims)
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
	idstr := ctx.Param("id")
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "参数错误",
		})
		h.l.Error("前端输入的 ID 不对", logger2.Error(err))
		return
	}
	uc := ctx.MustGet("claims").(*LUserClaims)
	var eg errgroup.Group
	var art domain.Article
	// 获取帖子的详细信息
	eg.Go(func() error {

		art, err = h.svc.GetPublishedById(ctx, id, uc.Uid)
		//if err != nil {
		//	ctx.JSON(http.StatusOK, Result{
		//		Code: 5,
		//		Msg:  "系统错误",
		//	})
		//	h.l.Error("获得文章信息失败", logger2.Error(err))
		//	return
		//}
		return err
	})
	var intr domain.Interactive
	//查询文件的阅读数 收藏数 点赞数之类的
	eg.Go(func() error {
		// 要在这里获得这篇文章的计数
		uc := ctx.MustGet("claims").(*LUserClaims)
		// 这个地方可以容忍错误
		intr, err = h.intrSvc.Get(ctx, h.biz, id, uc.Uid)
		// 这种是容错的写法
		//if err != nil {
		//	// 记录日志
		//}
		//return nil
		return err
	})
	// 在这儿等，要保证前面两个
	err = eg.Wait()
	if err != nil {
		// 代表查询出错了
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	//增加阅读计数 现在的做法是不是:只要redis 里面有才会更新redis 没有就不更新(只更新数据库) 等到有人读那个阅读数再加到redis里面
	// 增加阅读计数。
	go func() {
		// 开一个 goroutine，异步去执行
		er := h.intrSvc.IncrReadCnt(ctx, h.biz, art.Id)
		if er != nil {
			h.l.Error("增加阅读计数失败",
				logger2.Int64("aid", art.Id),
				logger2.Error(err))
		}
	}()

	// ctx.Set("art", art)

	// 这个功能是不是可以让前端，主动发一个 HTTP 请求，来增加一个计数？
	ctx.JSON(http.StatusOK, Result{
		Data: ArticleVO{
			Id:      art.Id,
			Title:   art.Title,
			Status:  art.Status.ToUint8(),
			Content: art.Content,
			// 要把作者信息带出去
			Author:     art.Author.Name,
			Ctime:      art.Ctime.Format(time.DateTime),
			Utime:      art.Utime.Format(time.DateTime),
			Liked:      intr.Liked,
			Collected:  intr.Collected,
			LikeCnt:    intr.LikeCnt,
			ReadCnt:    intr.ReadCnt,
			CollectCnt: intr.CollectCnt,
		},
	})
}

func (h *ArticleHandler) List(ctx *gin.Context, req ListReq, uc *jwt.UserClaims) (logger3.Result, error) {
	res, err := h.svc.List(ctx, uc.Uid, req.Offset, req.Limit)
	if err != nil {
		return logger3.Result{
			Code: 5,
			Msg:  "系统错误",
		}, nil
	}
	return logger3.Result{
		Data: slice.Map[domain.Article, ArticleVO](res,
			func(idx int, src domain.Article) ArticleVO {
				return ArticleVO{
					Id:       src.Id,
					Title:    src.Title,
					Abstract: src.Abstract(), //这个是摘要  如果是放在oss 的话 这个要另外存储在mysql 中
					Status:   src.Status.ToUint8(),
					// 这个列表请求，不需要返回内容
					//Content: src.Content,
					// 这个是创作者看自己的文章列表，也不需要这个字段
					//Author: src.Author
					Ctime: src.Ctime.Format(time.DateTime),
					Utime: src.Utime.Format(time.DateTime),
				}
			}),
	}, nil
}

func (h *ArticleHandler) Detail(ctx *gin.Context, usr jwt.UserClaims) (logger3.Result, error) {
	idstr := ctx.Param("id")
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		return logger3.Result{
			Code: 4,
			Msg:  "参数错误",
		}, err
	}
	art, err := h.svc.GetById(ctx, id)
	if err != nil {
		//ctx.JSON(http.StatusOK, )
		//a.l.Error("获得文章信息失败", logger.Error(err))
		return logger3.Result{
			Code: 5,
			Msg:  "系统错误",
		}, err
	}
	// 这是不借助数据库查询来判定的方法
	//要判断查询的文章作者 跟 当前登录信息的用户id 是否一致
	if art.Author.Id != usr.Uid {
		return logger3.Result{
			Code: 5,
			Msg:  "用户信息异常",
		}, fmt.Errorf("非法访问文章，创作者 ID 不匹配 %d", usr.Uid)
	}
	return logger3.Result{
		Data: ArticleVO{
			Id:    art.Id,
			Title: art.Title,
			// 不需要这个摘要信息
			//Abstract: art.Abstract(),
			Status:  art.Status.ToUint8(),
			Content: art.Content,
			// 这个是创作者看自己的文章列表，也不需要这个字段
			//Author: art.Author
			Ctime: art.Ctime.Format(time.DateTime),
			Utime: art.Utime.Format(time.DateTime),
		},
	}, nil
}

func (h *ArticleHandler) Like(ctx *gin.Context, request LikeReq, uc *jwt.UserClaims) (logger3.Result, error) {
	var err error
	if request.Like {
		err = h.intrSvc.Like(ctx, h.biz, request.Id, uc.Uid)
	} else {
		err = h.intrSvc.CancelLike(ctx, h.biz, request.Id, uc.Uid)
	}

	if err != nil {
		return logger3.Result{
			Code: 5,
			Msg:  "系统错误",
		}, err
	}
	return logger3.Result{Msg: "OK"}, nil
}

func (h *ArticleHandler) GetTop(ctx *gin.Context) {
	//可以前端参数传key
	key := ctx.Param("key")
	top := ctx.Param("top")
	var topN int64
	if key == "" {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "参数异常",
			Data: nil,
		})
		return
	}
	if top == "" {
		topN = 9
	} else {
		topN = gocast.Int64(top)
	}

	//repo   repository.InteractiveRepository
	//直接获取排名
	interactives, err := h.intrSvc.GetTopN(ctx, key, topN)
	if err != nil {
		h.l.Error("查询topN 失败")
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "查询失败",
			Data: nil,
		})
		return
	}
	//然后根据 biz_id + biz 找到对应的帖子的作者以及详细内容

	//响应到前端
	h.l.Debug("查询成功")
	ctx.JSON(http.StatusOK, Result{
		Code: 200,
		Msg:  "查询成功",
		Data: interactives,
	})
	return
}
