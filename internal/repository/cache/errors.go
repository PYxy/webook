package cache

import "errors"

var (
	ErrUnknown                = errors.New("未知错误")
	ErrFrequentlyForSend      = errors.New("请求频率过于频繁")
	ErrUnknownForCode         = errors.New("code异常")
	ErrCodeVerifyTooManyTimes = errors.New("验证码一直输入错误")
	ErrAttack                 = errors.New("受到攻击")
)
