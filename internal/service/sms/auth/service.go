package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

type AuthSMSService struct {
	svc sms.Service
	key string //解密的密钥
}

// Send
//biz
//1.是由线下给他的加密过的 字符串
//2.或者通过请求接口返回一个
//func (s *AuthSMSService) GenerateToken(ctx context.Context, tplId string) (string, error) {
//
//}

func NewAuthSMSService(svc sms.Service, key string) sms.Service {
	return &AuthSMSService{
		svc: svc,
		key: key,
	}
}

// Send  args 里面应该是有一个带业务标识的参数 就是下面的tpl
func (a *AuthSMSService) Send(ctx context.Context, biz string, phoneNumbers []string, args []sms.ArgVal) error {
	//TODO implement me
	var claims Claims
	token, err := jwt.ParseWithClaims(biz, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.key), nil
	})

	//biz
	if err != nil {
		return err
	}
	if !token.Valid {
		return errors.New("token 不合法")
	}
	//可以根据Tpl 限流或者往下传

	return a.svc.Send(ctx, claims.Tpl, phoneNumbers, args)
}

type Claims struct {
	jwt.RegisteredClaims
	Tpl string //业务类型
}
