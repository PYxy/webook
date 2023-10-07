package logger3

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type AppHandleError[T any] func(ctx *gin.Context, req T) (Result, error)

func InnerWarp[T any](handler AppHandleError[T]) gin.HandlerFunc {

	return func(ctx *gin.Context) {

	}
}

func NewFunc[T any](func(ctx *gin.Context)) AppHandleError[T] {
	return func(ctx *gin.Context, req T) (Result, error) {
		var (
			res Result
			err error
		)

		return res, err
	}
}

func WrapReq[T any](fn func(ctx *gin.Context, req T) (Result, error)) gin.HandlerFunc {

	return func(ctx *gin.Context) {
		var req T

		if err := ctx.Bind(&req); err != nil {
			//打印日志
			return
		}
		res, err := fn(ctx, req)
		if err != nil {
			//打印日志
			return
		}

		ctx.JSON(http.StatusOK, res)
	}
}

type Result struct {
}
