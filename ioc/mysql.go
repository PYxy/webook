package ioc

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gitee.com/geekbang/basic-go/webook/config"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
)

func InitMysql() *gorm.DB {
	// 3s 1m 1h
	//viper.GetDuration()
	//没有也不会报错, 会返回类型的零值
	dsn := viper.GetString("db.mysql.dsn")
	fmt.Println(dsn)
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	var cfg Config
	//这里可以设置默认值，即使配置文件没有db.mysql 下面没有DSN 配置项
	//适用
	/*
		redis:
		  dsn: "webook-redis-service:6380"
	*/
	//不适用  会显示为 ""
	/*
		db.mysql:
		  dsn: ""

		redis:
		  dsn: "webook-redis-service:6380"
	*/
	cfg.DSN = "root:root@tcp(127.0.0.1:3306)/webook_default"
	err := viper.UnmarshalKey("db.mysql", &cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(cfg)

	//dsn 获取为空 它都不会报错的 所有需要做一些默认值的设置
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", config.Config.DB.DSN)))
	if err != nil {
		// 我只会在初始化过程中 panic
		// panic 相当于整个 goroutine 结束
		// 一旦初始化过程出错，应用就不要启动了
		panic(err)
	}

	dao.NewUserDAOv1(func() *gorm.DB {
		// 动态获取db 配置
		viper.OnConfigChange(func(in fsnotify.Event) {
			dsn = viper.GetString("mysql.dsn")
			newdb, err := gorm.Open(mysql.Open(dsn))
			if err != nil {
				fmt.Println("动态获取新配置文件时,mysql 初始化失败")
			}
			pt := unsafe.Pointer(db)

			atomic.StorePointer(&pt, unsafe.Pointer(newdb))

		})
		//要用原子操作
		//pt := unsafe.Pointer(&db)
		//val :=atomic.LoadPointer(&pt)
		//return (*gorm.DB)(val)
		return db
	})
	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}
