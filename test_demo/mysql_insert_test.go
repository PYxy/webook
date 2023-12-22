package test_demo

import (
	"fmt"
	slog "log"
	"os"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type User12 struct {
	gorm.Model
	Name string `gorm:"unique;not null"` // 设置 memberNumber 字段唯一且不为空
	//Age      sql.NullInt64
	//Birthday *time.Time
	Email string `gorm:"type:varchar(90);unique_index"`
	Role  string `gorm:"size:255"` //设置字段的大小为255个字节

	Num      int    `gorm:"AUTO_INCREMENT"` // 设置 Num字段自增
	Address  string `gorm:"index:addr"`     // 给Address 创建一个名字是  `addr`的索引
	IgnoreMe int    `gorm:"-"`              //忽略这个字段
}

func Test_insert(t *testing.T) {
	db, err := gorm.Open(mysql.Open("root:r4t7u#8i9s@tcp(120.132.118.90:3306)/test?charset=utf8&parseTime=True&loc=Local&timeout=5s"), &gorm.Config{
		Logger: logger.New(
			slog.New(os.Stdout, "\r\n", slog.LstdFlags), // io writer
			logger.Config{
				SlowThreshold: time.Second, // 慢查询 SQL 阈值
				Colorful:      false,       // 禁用彩色打印
				//IgnoreRecordNotFoundError: false,
				LogLevel: logger.Info, // Log lever
			},
		),
		//Logger:                                   sqlLog, //可以使用一个全局变量来定义是不是打印日志
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 表名是否加 s
		},
	})
	fmt.Println(err)
	db.AutoMigrate(&User12{})
	var srcIds []User12
	for i := 0; i < 5; i++ {
		srcIds = append(srcIds, User12{
			Name:     fmt.Sprintf("name:%d", i+1),
			Email:    fmt.Sprintf("120615985%d.163.com", i),
			Role:     "root",
			Num:      0,
			Address:  "xxx",
			IgnoreMe: 0,
		})
	}
	//rows, err := db.Model(&User12{}).Limit(1).Rows()
	//if err != nil {
	//	panic(err)
	//}
	//直接批量插入 有就更新 没有就插入
	//columns, err := rows.Columns()
	//err = db.Clauses(&clause.OnConflict{
	//
	//	// 我们需要 Entity 告诉我们，修复哪些数据
	//	//无论columns  有多少个字段 都是安装 字段中的唯一字段去更新
	//	DoUpdates: clause.AssignmentColumns(columns),
	//}).Create(&srcIds).Error

	targetId := []uint{1, 2, 3, 4, 12, 14, 16, 18, 20}
	var mysqlSearch []User12
	err = db.Model(&User12{}).Find(&mysqlSearch).Error
	fmt.Println(err)
	compare_slice := slice.Map(mysqlSearch, func(idx int, src User12) uint {
		return src.ID
	})
	fmt.Println(targetId)
	fmt.Println(compare_slice)
	fmt.Println(slice.DiffSet(targetId, compare_slice))

}
