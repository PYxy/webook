////go:build live

package fixer

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gitee.com/geekbang/basic-go/webook/pkg/migrator"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator/events"
)

// 这个是课堂演示
type Fixer[T migrator.Entity] struct {
	base    *gorm.DB
	target  *gorm.DB
	columns []string
}

// 最一了百了的写法
// 不管三七二十一，我TM直接覆盖
// 把 event 当成一个触发器，不依赖的 event 的具体内容（ID 必须不可变）
// 修复这里，也改成批量？？
func (f *Fixer[T]) Fix(ctx context.Context, evt events.InconsistentEvent) error {
	var t T
	err := f.base.WithContext(ctx).
		Where("id =?", evt.ID).First(&t).Error
	switch err {
	case nil:
		// base 有数据
		// 修复数据的时候，可以考虑增加 WHERE base.utime >= target.utime
		// utime 用不了，就看有没有version 之类的，或者能够判定数据新老的
		return f.target.WithContext(ctx).
			Clauses(clause.OnConflict{
				DoUpdates: clause.AssignmentColumns(f.columns),
			}).Create(&t).Error
	case gorm.ErrRecordNotFound:
		// base 没了
		return f.target.WithContext(ctx).
			Where("id=?", evt.ID).Delete(&t).Error
	default:
		return err
	}
}

// base 和 target 在校验时候的数据，到你修复的时候就变了
func (f *Fixer[T]) FixV1(ctx context.Context, evt events.InconsistentEvent) error {
	switch evt.Type {
	case events.InconsistentEventTypeTargetMissing,
		events.InconsistentEventTypeNEQ:
		// 这边要插入
		var t T
		err := f.base.WithContext(ctx).
			Where("id =?", evt.ID).First(&t).Error
		switch err {
		case gorm.ErrRecordNotFound:
			// base 也删除了这条数据
			return f.target.WithContext(ctx).
				Where("id=?", evt.ID).Delete(new(T)).Error
		case nil:
			return f.target.Clauses(clause.OnConflict{
				// 这边要更新全部列
				DoUpdates: clause.AssignmentColumns(f.columns),
			}).Create(&t).Error
		default:
			return err
		}
		// 这边要更新
	case events.InconsistentEventTypeBaseMissing:
		return f.target.WithContext(ctx).
			Where("id=?", evt.ID).Delete(new(T)).Error
	default:
		return errors.New("未知的不一致类型")
	}
}

// 一定要抓住，base 在校验时候的数据，到你修复的时候就变了
func (f *Fixer[T]) FixV2(ctx context.Context, evt events.InconsistentEvent) error {
	switch evt.Type {
	case events.InconsistentEventTypeTargetMissing:
		// 这边要插入
		var t T
		err := f.base.WithContext(ctx).
			Where("id =?", evt.ID).First(&t).Error
		switch err {
		case gorm.ErrRecordNotFound:
			// base 也删除了这条数据
			return nil
		case nil:
			// 就在你插入的时候，双写的程序，也插入了，你就会冲突
			return f.target.Create(&t).Error
		default:
			return err
		}
	case events.InconsistentEventTypeNEQ:
		var t T
		err := f.base.WithContext(ctx).
			Where("id =?", evt.ID).First(&t).Error
		switch err {
		case gorm.ErrRecordNotFound:
			// target 要删除
			return f.target.WithContext(ctx).
				Where("id=?", evt.ID).Delete(&t).Error
		case nil:
			return f.target.Updates(&t).Error
		default:
			return err
		}
		// 这边要更新
	case events.InconsistentEventTypeBaseMissing:
		return f.target.WithContext(ctx).
			Where("id=?", evt.ID).Delete(new(T)).Error
	default:
		return errors.New("未知的不一致类型")
	}
}
