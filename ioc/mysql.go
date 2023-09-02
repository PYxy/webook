package ioc

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gitee.com/geekbang/basic-go/webook/config"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
)

func InitMysql() *gorm.DB {
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", config.Config.DB.DSN)))
	if err != nil {
		// 我只会在初始化过程中 panic
		// panic 相当于整个 goroutine 结束
		// 一旦初始化过程出错，应用就不要启动了
		panic(err)
	}
	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}
