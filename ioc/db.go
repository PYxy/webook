package ioc

import "gorm.io/gorm"

// SrcDB 纯粹是为了 wire 而准备的
type SrcDB *gorm.DB

// DstDB 纯粹是为了 wire 而准备的
type DstDB *gorm.DB
