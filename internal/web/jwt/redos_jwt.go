package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	AccessKey  = []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0")
	RefreshKey = []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf1")
)

type RedisJWTHandler struct {
	cmd redis.Cmdable
}

func NewRedisJWTHandler(cmd redis.Cmdable) Handler {
	return &RedisJWTHandler{cmd: cmd}
}

func (r *RedisJWTHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	//TODO implement me
	ssid := uuid.New().String()
	err := r.SetJWTToken(ctx, uid, ssid)
	if err != nil {
		return err
	}

	err = r.setRefreshToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisJWTHandler) setRefreshToken(ctx *gin.Context, uid int64, ssid string) error {
	claims := &RefreshClaims{
		Ssid: ssid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
		Uid: uid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString(RefreshKey)
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}

func (r *RedisJWTHandler) SetJWTToken(ctx *gin.Context, uid int64, ssid string) error {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
		Uid:       uid,
		Ssid:      ssid,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString(AccessKey)
	if err != nil {
		return err
	}
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

func (r *RedisJWTHandler) ClearToken(ctx *gin.Context) error {
	//TODO implement me
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")
	claims := ctx.MustGet("claims").(*UserClaims)
	//过期时间以 refresh-token 的过期时间为准
	return r.cmd.Set(ctx, fmt.Sprintf("users:ssid:%s", claims.Ssid), "", time.Hour*24*7).Err()

}

func (r *RedisJWTHandler) CheckSession(ctx *gin.Context, ssid string) error {
	//TODO implement me
	err := r.cmd.Get(ctx, fmt.Sprintf("users:ssid:%s", ssid)).Err()
	//有 返回nil
	//没有 返回 redis.Nil
	// 还有就是redis 异常
	fmt.Printf("ssid %s 是否存在: %v \n", fmt.Sprintf("users:ssid:%s", ssid), err)
	return err

}

func (r *RedisJWTHandler) ExtractToken(ctx *gin.Context) string {
	// JWT 来校验
	tokenHeader := ctx.GetHeader("Authorization")
	segs := strings.Split(tokenHeader, " ")
	if len(segs) != 2 {
		return ""
	}
	return segs[1]
}
