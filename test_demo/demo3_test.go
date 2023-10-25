package test_demo

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DBProvider func() *gorm.DB

var Gdb2 *gorm.DB

type User struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 全部用户唯一
	Email    sql.NullString `gorm:"unique"`
	Phone    sql.NullString `gorm:"unique"`
	Password string
	// 往这面加
	// 创建时间，毫秒数
	Ctime int64
	// 更新时间，毫秒数
	Utime int64
	//其实现在有优化 只要用到覆盖索引 都有机会使用联合索引或者普通的二级索引
	// 如果要创建联合索引，<unionid, openid>，用 openid 查询的时候不会走索引
	// <openid, unionid> 用 unionid 查询的时候，不会走索引
	// 微信的字段
	WechatUnionID sql.NullString
	WechatOpenID  sql.NullString `gorm:"unique"`
	//昵称
	NickName string
	//生日
	BirthDay string
	//个人描述
	Describe string
}

type DBProvider2 func(db unsafe.Pointer) *gorm.DB

type UserDAO2 struct {
	db *gorm.DB
	//动态监控配置文件的变更
	p DBProvider
}

func (u *UserDAO2) Get(ctx context.Context) error {
	var aa User
	err := u.p().WithContext(ctx).Where("id = ?", 4).First(&aa).Error
	fmt.Println(aa)
	fmt.Println(err)
	return err
}

type UserDaoInterface2 interface {
	Get(ctx context.Context) error
}

func (u *UserDAO2) SetUserDAO(p DBProvider) UserDaoInterface2 {
	//法1
	u.p = p
	return u

}

func NewUserDAO2(db *gorm.DB, p DBProvider) UserDaoInterface2 {
	//法1
	u := &UserDAO2{
		db: db,
	}
	//动态加载配置文件
	u.SetUserDAO(p)
	return u
}

func NewDB2() {
	cfile := pflag.String("config", "./dev.yaml", "指定配置文件的路径")
	pflag.Parse()

	viper.SetConfigFile(*cfile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	dsn := viper.GetString("db.mysql.dsn")
	fmt.Println("mysql 连接地址:", dsn)
	Gdb2, err = gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", dsn)))
	if err != nil {
		// 我只会在初始化过程中 panic
		// panic 相当于整个 goroutine 结束
		// 一旦初始化过程出错，应用就不要启动了
		panic(err)
	}
	viper.WatchConfig()
	//开启之后 修改物理配置文件1
	//viper.OnConfigChange(func(in fsnotify.Event) {
	//	//只能知道变化了 但是 不知道那个数据发生变化了,只能重新读一次对应使用的配置
	//	//如
	//	fmt.Println("老DB对象:", db)
	//	dsn = viper.GetString("db.mysql.dsn")
	//	fmt.Println("发生变化了:", dsn)
	//	fmt.Println(in.Name, in.Op)
	//	newdb, err := gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", dsn)))
	//	fmt.Println("新创建连接的db对象:", newdb)
	//	atomic.StorePointer(&GDB, unsafe.Pointer(newdb))
	//	fmt.Println(err)
	//	fmt.Println("设置成功1..")
	//	fmt.Println("设置成功后db对象:", (*gorm.DB)(GDB))
	//})

	viper.OnConfigChange(func(in fsnotify.Event) {
		//只能知道变化了 但是 不知道那个数据发生变化了,只能重新读一次对应使用的配置
		//如
		fmt.Println("老DB对象:", Gdb2)
		dsn = viper.GetString("db.mysql.dsn")
		fmt.Println("发生变化了:", dsn)
		fmt.Println(in.Name, in.Op)
		newdb, err := gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", dsn)))
		fmt.Println("新创建连接的db对象:", newdb)
		oldDB := unsafe.Pointer(Gdb2)
		atomic.StorePointer(&oldDB, unsafe.Pointer(newdb))
		fmt.Println(err)
		fmt.Println("设置成功1..")
		fmt.Println("替换后的db对象:", Gdb2)
	})

}

func Test_b(t *testing.T) {
	//读取配置文件
	NewDB2()
	a := NewUserDAO2(Gdb2,
		func() *gorm.DB {
			a := unsafe.Pointer(Gdb2)
			//fmt.Println("unsafe操作获取到的1:", (*gorm.DB)(atomic.LoadPointer(&a)))

			return (*gorm.DB)(atomic.LoadPointer(&a))

		})

	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < 1; i++ {
		go func(ctx context.Context) {

			for {
				if err := a.Get(ctx); err != nil {
					return
				}
				time.Sleep(time.Second * 2)

			}

		}(ctx)
	}
	//err := a.Get(ctx)
	//fmt.Println(err)

	time.Sleep(time.Second * 30)
	cancel()
}
