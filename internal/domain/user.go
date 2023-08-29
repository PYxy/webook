package domain

import (
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
	Ctime    time.Time
}
