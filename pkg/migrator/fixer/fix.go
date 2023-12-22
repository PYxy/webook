package fixer

import (
	"context"

	"github.com/ecodeclub/ekit/slice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gitee.com/geekbang/basic-go/webook/pkg/migrator"
)

type OverrideFixer[T migrator.Entity] struct {
	// 因为本身其实这个不涉及什么领域对象，
	// 这里操作的不是 migrator 本身的领域对象
	base    *gorm.DB
	target  *gorm.DB
	columns []string
}

func NewOverrideFixer[T migrator.Entity](base *gorm.DB,
	target *gorm.DB) (*OverrideFixer[T], error) {
	// 在这里需要查询一下数据库中究竟有哪些列
	var t T
	rows, err := base.Model(&t).Limit(1).Rows()
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	return &OverrideFixer[T]{
		base:    base,
		target:  target,
		columns: columns,
	}, nil
}

func (o *OverrideFixer[T]) Fix(ctx context.Context, id int64) error {
	var src T
	// 找出数据
	err := o.base.WithContext(ctx).Where("id = ?", id).
		First(&src).Error
	switch err {
	// 找到了数据
	case nil:

		return o.target.Clauses(&clause.OnConflict{
			// 我们需要 Entity 告诉我们，修复哪些数据
			DoUpdates: clause.AssignmentColumns(o.columns),
		}).Create(&src).Error
	case gorm.ErrRecordNotFound:
		return o.target.Delete("id = ?", id).Error
	default:
		return err
	}
}

func (o *OverrideFixer[T]) FixInBatches(ctx context.Context, idSlice []int64) error {
	var srcSlice []T

	//传过来的数据 有一些还在  有些可能已经删除了
	// 找出数据
	err := o.base.WithContext(ctx).Where("id in ?", idSlice).
		First(&srcSlice).Error
	if err != nil {
		return err
	}
	// 查询base 中还有的数据
	srcIds := slice.Map(srcSlice, func(idx int, t T) int64 {
		return t.ID()
	})
	// 对比 获取base 中已经删除的数据
	diff := slice.DiffSet(idSlice, srcIds)
	if len(srcIds) >= 0 {
		err = o.target.Clauses(&clause.OnConflict{
			// 我们需要 Entity 告诉我们，修复哪些数据
			DoUpdates: clause.AssignmentColumns(o.columns),
		}).Create(&srcIds).Error
		if err != nil {
			return err
		}
	}
	// 删除 base 数据库中没有的数据
	if len(diff) > 0 {
		return o.target.Delete("id in ?", diff).Error
	}
	return nil

}
