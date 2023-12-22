package connpool

import (
	"context"
	"database/sql"
	"errors"
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"gorm.io/gorm"
)

var errUnknownPattern = errors.New("未知的双写 pattern")

type DoubleWritePool struct {
	pattern *atomicx.Value[string]
	src     gorm.ConnPool
	dst     gorm.ConnPool
}

func NewDoubleWritePool(srcDB *gorm.DB, dst *gorm.DB) *DoubleWritePool {
	return &DoubleWritePool{
		src:     srcDB.ConnPool,
		dst:     dst.ConnPool,
		pattern: atomicx.NewValueOf(PatternSrcOnly)}
}

func (d *DoubleWritePool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	pattern := d.pattern.Load()
	switch pattern {
	case PatternSrcOnly:
		tx, err := d.src.(gorm.TxBeginner).BeginTx(ctx, opts)
		return &DoubleWriteTx{
			pattern: pattern,
			src:     tx,
		}, err
	case PatternSrcFirst:
		return d.startTwoTx(d.src, d.dst, pattern, ctx, opts)
	case PatternDstFirst:
		return d.startTwoTx(d.src, d.dst, pattern, ctx, opts)
	case PatternDstOnly:
		tx, err := d.dst.(gorm.TxBeginner).BeginTx(ctx, opts)
		return &DoubleWriteTx{
			pattern: pattern,
			src:     tx,
		}, err
	default:
		return nil, errUnknownPattern
	}
}

func (d *DoubleWritePool) startTwoTx(
	first gorm.ConnPool,
	second gorm.ConnPool,
	pattern string,
	ctx context.Context,
	opts *sql.TxOptions) (*DoubleWriteTx, error) {
	src, err := first.(gorm.TxBeginner).BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	dst, err := second.(gorm.TxBeginner).BeginTx(ctx, opts)
	if err != nil {
		// 记录日志
		_ = src.Rollback()
	}
	return &DoubleWriteTx{src: src, dst: dst, pattern: pattern}, nil
}

func (d *DoubleWritePool) ChangePattern(pattern string) {
	d.pattern.Store(pattern)
}

func (d *DoubleWritePool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	// 禁用这个东西，因为我们没有办法创建出来 sql.Stmt 实例
	panic("implement me")
}

func (d *DoubleWritePool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	switch d.pattern.Load() {
	case PatternSrcOnly:
		return d.src.ExecContext(ctx, query, args...)
	case PatternSrcFirst:
		res, err := d.src.ExecContext(ctx, query, args...)
		if err == nil {
			_, err1 := d.dst.ExecContext(ctx, query, args...)
			if err1 != nil {
				// 这边要记录日志
				// 并且要通知修复数据
			}
		}
		return res, err
	case PatternDstFirst:
		res, err := d.dst.ExecContext(ctx, query, args...)
		if err == nil {
			_, err1 := d.src.ExecContext(ctx, query, args...)
			if err1 != nil {
				// 这边要记录日志
				// 并且要通知修复数据
			}
		}
		return res, err
	case PatternDstOnly:
		return d.dst.ExecContext(ctx, query, args...)
	default:
		return nil, errUnknownPattern
	}
}

func (d *DoubleWritePool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	switch d.pattern.Load() {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryContext(ctx, query, args...)
	case PatternDstFirst, PatternDstOnly:
		return d.dst.QueryContext(ctx, query, args...)
	default:
		return nil, errUnknownPattern
	}
}

func (d *DoubleWritePool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	switch d.pattern.Load() {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryRowContext(ctx, query, args...)
	case PatternDstFirst, PatternDstOnly:
		return d.dst.QueryRowContext(ctx, query, args...)
	default:
		// 因为返回值里面咩有 error，只能 panic 掉
		panic(errUnknownPattern)
	}
}

type DoubleWriteTx struct {
	pattern string
	src     *sql.Tx
	dst     *sql.Tx
}

func (d *DoubleWriteTx) Commit() error {
	switch d.pattern {
	case PatternSrcFirst:
		err := d.src.Commit()
		if d.dst != nil {
			err1 := d.dst.Commit()
			if err1 != nil {
				// 记录日志
			}
		}
		return err
	case PatternSrcOnly:
		return d.src.Commit()
	case PatternDstFirst:
		err := d.dst.Commit()
		if d.src != nil {
			err1 := d.src.Commit()
			if err1 != nil {
				// 记录日志
			}
		}

		return err
	case PatternDstOnly:
		return d.dst.Commit()
	default:
		return errUnknownPattern
	}
}

func (d *DoubleWriteTx) Rollback() error {
	switch d.pattern {
	case PatternSrcFirst:
		err := d.src.Rollback()
		if d.dst != nil {
			err1 := d.dst.Rollback()
			if err1 != nil {
				// 记录日志
			}
		}
		return err
	case PatternSrcOnly:
		return d.src.Rollback()
	case PatternDstOnly:
		return d.dst.Rollback()
	case PatternDstFirst:
		err := d.dst.Rollback()
		if d.src != nil {
			err1 := d.src.Rollback()
			if err1 != nil {
				// 记录日志
			}
		}
		return err
	default:
		return errUnknownPattern
	}
}

func (d *DoubleWriteTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	panic("implement me")
}

func (d *DoubleWriteTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	switch d.pattern {
	case PatternSrcOnly:
		return d.src.ExecContext(ctx, query, args...)
	case PatternSrcFirst:
		res, err := d.src.ExecContext(ctx, query, args...)
		if err == nil {
			_, err1 := d.dst.ExecContext(ctx, query, args...)
			if err1 != nil {
				// 这边要记录日志
				// 并且要通知修复数据
			}
		}
		return res, err
	case PatternDstFirst:
		res, err := d.dst.ExecContext(ctx, query, args...)
		if err == nil {
			_, err1 := d.src.ExecContext(ctx, query, args...)
			if err1 != nil {
				// 这边要记录日志
				// 并且要通知修复数据
			}
		}
		return res, err
	case PatternDstOnly:
		return d.dst.ExecContext(ctx, query, args...)
	default:
		return nil, errUnknownPattern
	}
}

func (d *DoubleWriteTx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	switch d.pattern {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryContext(ctx, query, args...)
	case PatternDstFirst, PatternDstOnly:
		return d.dst.QueryContext(ctx, query, args...)
	default:
		return nil, errUnknownPattern
	}
}

func (d *DoubleWriteTx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	switch d.pattern {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryRowContext(ctx, query, args...)
	case PatternDstFirst, PatternDstOnly:
		return d.dst.QueryRowContext(ctx, query, args...)
	default:
		// 因为返回值里面咩有 error，只能 panic 掉
		panic(errUnknownPattern)
	}
}

const (
	PatternSrcOnly  = "src_only"
	PatternSrcFirst = "src_first"
	PatternDstFirst = "dst_first"
	PatternDstOnly  = "dst_only"
)
