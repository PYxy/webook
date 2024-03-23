package k6

import (
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

func TestHello(t *testing.T) {
	server := gin.Default()
	// 压测这个接口
	server.POST("/hello", func(ctx *gin.Context) {
		var u User
		ctx.Bind(&u)
		r := rand.Int31n(1000)
		// sleep 随机时间，来模拟业务的执行时间
		time.Sleep(time.Millisecond * time.Duration(r))
		// 这里我们模拟一下错误
		// 模拟 10% 比例的错误
		//if r%100 < 10 {
		//	// 返回了 5xx，模拟失败
		//	ctx.String(http.StatusInternalServerError, "系统错误")
		//} else {
		//	// 返回了 200，模拟成功
		//	ctx.String(http.StatusOK, u.Name)
		//}
		ctx.String(http.StatusOK, u.Name)
	})
	server.Run(":8080")
}

type User struct {
	Name string `json:"name"`
}
