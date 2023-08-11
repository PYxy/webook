package dao

import "gorm.io/gorm"

func InitTable(db *gorm.DB) error {
	return db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").AutoMigrate(&User{})
}
