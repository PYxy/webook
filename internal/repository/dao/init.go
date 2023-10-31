package dao

import (
	"gorm.io/gorm"

	"gitee.com/geekbang/basic-go/webook/internal/repository/dao/article"
)

func InitTable(db *gorm.DB) error {
	return db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").AutoMigrate(&User{}, &article.Article{},
		&article.PublishedArticle{},
		&Interactive{},
		&UserLikeBiz{},
		&Collection{},
		&UserCollectionBiz{})
}
