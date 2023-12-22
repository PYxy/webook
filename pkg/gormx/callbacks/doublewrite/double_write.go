package doublewrite

import (
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"gorm.io/gorm"
)

type Callback struct {
	pattern *atomicx.Value[string]
}

func (c *Callback) doubleWriteCreate() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		// 这里有一个固定的 db，只能是 src 或者 dst，
		// 没有办法切换，也灭有办法中断
	}
}
