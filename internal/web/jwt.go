package web

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT interface {
	Encryption(map[string]string) (accessToken string, refreshToken string, err error)
	Decrypt(tokenStr string) (interface{}, error)
}

type Jwt struct {
	secretCode string
}

func (j *Jwt) Encryption(arg map[string]string) (accessToken string, refreshToken string, err error) {
	now := time.Now()
	fingerprint, ok := arg["fingerprint"]
	if !ok {
		return "", "", errors.New("参数缺失")
	}
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 30)),
		},
		Fingerprint: fingerprint,
	}
	//access_token 被用来正常访问数据
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	accessToken, err = token.SignedString([]byte(j.secretCode))
	if err != nil {
		return "", "", err
	}
	//ctx.Header("x-access-token", accessToken)
	//refresh_token 用来刷新 access_token。
	//修改 refresh_token 的过期时间
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Hour * 168))
	token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	//下面的密钥可以使用不同的密钥(一样的也行)
	refreshToken, err = token.SignedString([]byte(j.secretCode))
	if err != nil {
		//删除 access-token
		return "", "", err
	}
	//可以换一种方式保持到redis里面,避免refresh_token 被人拿到之后一直使用
	//可以使用MD5 转一下,或者直接截取指定长度的字符串 如: 以key 为 前面获取到的字符串

	return accessToken, refreshToken, nil
}

func (j *Jwt) Decrypt(tokenStr string) (interface{}, error) {
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		//[]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0") 要给后面接口中的一致
		//95osj3fUD7fo0mlYdDbncXz4VD2igvf0"
		return []byte(j.secretCode), nil
	})
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		//解析成功  但是 token 以及 claims 不一定合法
		return nil, errors.New("不合法操作")
	}
	return claims, nil
}
