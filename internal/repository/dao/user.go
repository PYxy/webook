package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

var (
	ErrUserDuplicate = errors.New("数据冲突")
	ErrUserNotFound  = gorm.ErrRecordNotFound
)

type DBProvider func() *gorm.DB

type UserDAO struct {
	db *gorm.DB

	//动态监控配置文件的变更
	p DBProvider
}

func NewUserDAOv1(p DBProvider) UserDaoInterface {
	//法1
	return &UserDAO{
		p: p,
	}
	
}

func NewUserDAO(db *gorm.DB) UserDaoInterface {
	//法1
	return &UserDAO{
		db: db,
	}

	//法2 动态获取db 配置
	//res := &UserDAO{db: db}
	//viper.OnConfigChange(func(in fsnotify.Event) {
	//	db, err := gorm.Open(mysql.Open())
	//	pt := unsafe.Pointer(&res.db)
	//	atomic.StorePointer(&pt, unsafe.Pointer(&db))
	//})
	//return res
}

//func (dao *UserDAO) FindById(ctx context.Context, id int64) (User, error) {
//	var u User
//	err := dao.db.WithContext(ctx).Where("`id` = ?", id).First(&u).Error
//	return u, err
//}

func (dao *UserDAO) FindByWeChat(ctx context.Context, openId string) (u User, err error) {
	////动态监控配置文件的变更
	//err = dao.p().WithContext(ctx).Where("wechat_open_id = ?", openId).First(&u).Error

	err = dao.db.WithContext(ctx).Where("wechat_open_id = ?", openId).First(&u).Error
	return u, err
}

func (dao *UserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	//err := dao.db.WithContext(ctx).First(&u, "email = ?", email).Error
	//gorm.ErrRecordNotFound
	return u, err
}

func (dao *UserDAO) Insert(ctx context.Context, u User) error {
	// 存毫秒数
	now := time.Now().UnixMilli()
	u.Utime = now
	u.Ctime = now
	err := dao.db.WithContext(ctx).Create(&u).Error
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const uniqueConflictsErrNo uint16 = 1062
		if mysqlErr.Number == uniqueConflictsErrNo {
			// 邮箱冲突
			return ErrUserDuplicate
		}
	}
	return err
}

func (dao *UserDAO) Update(ctx context.Context, u User) error {
	// 存毫秒数
	now := time.Now().UnixMilli()
	u.Utime = now
	fmt.Println("需要更新的id:", u.Id)
	err := dao.db.WithContext(ctx).Model(&User{}).Where(&User{Id: u.Id}).Updates(u).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUserNotFound
	}
	return err

}

func (dao *UserDAO) FindById(ctx context.Context, id int64) (u User, err error) {
	err = dao.db.WithContext(ctx).Model(&User{}).Where(&User{Id: id}).First(&u).Error
	return
}

func (dao *UserDAO) FindByPhone(ctx context.Context, phone string) (u User, err error) {
	err = dao.db.WithContext(ctx).Model(&User{}).Where("phone = ?", phone).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return u, ErrUserNotFound
	}
	return u, err
}

// User 直接对应数据库表结构
// 有些人叫做 entity，有些人叫做 model，有些人叫做 PO(persistent object)
// PO是持久化对象，用于表示数据库中的一条记录映射成的对象，类中应该都是基本数据类型和String，而不是更复杂的类型，因为要和数据库表字段对应
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
