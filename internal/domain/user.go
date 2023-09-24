package domain

import (
	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
	"time"
)

// User 领域对象，是 DDD 中的 entity
// BO(business object)
// BO是实际的业务对象，会参与业务逻辑的处理操作，里面可能会包含多个类，用于表示一个业务对象
type User struct {
	Id       int64
	Email    string
	Phone    string
	Password string
	NickName string
	BirthDay string
	Describe string
	// 不要组合，万一你将来可能还有 DingDingInfo，里面有同名字段 UnionID
	WechatInfo WechatInfo
	Ctime      time.Time
}

type SMSBO struct {
	Id           int
	Biz          string
	PhoneNumbers []string
	Args         []sms.ArgVal
}
