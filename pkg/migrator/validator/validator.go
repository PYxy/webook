package validator

import (
	"context"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator/events"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"time"
)

// Validator T 必须实现了 Entity 接口
type Validator[T migrator.Entity] struct {
	// 校验，以 XXX 为准，
	base *gorm.DB
	// 校验的是谁的数据
	target *gorm.DB
	l      logger.LoggerV1

	p events.Producer

	direction string

	batchSize int

	highLoad *atomicx.Value[bool]

	// 在这里加字段，比如说，在查询 base 根据什么列来排序，在 target 的时候，根据什么列来查询数据
	// 最极端的情况，是这样

	utime int64
	// <=0 说明直接退出校验循环
	// > 0 真的 sleep
	sleepInterval time.Duration

	fromBase func(ctx context.Context, offset int) (T, error)
}

func NewValidator[T migrator.Entity](
	base *gorm.DB,
	target *gorm.DB,
	direction string,
	l logger.LoggerV1,
	p events.Producer) *Validator[T] {
	highLoad := atomicx.NewValueOf[bool](false)
	go func() {
		// 在这里，去查询数据库的状态
		// 你的校验代码不太可能是性能瓶颈，性能瓶颈一般在数据库
		// 你也可以结合本地的 CPU，内存负载来判定
	}()
	res := &Validator[T]{base: base, target: target,
		l: l, p: p, direction: direction,
		highLoad: highLoad}
	res.fromBase = res.fullFromBase
	return res
}

func (v *Validator[T]) SleepInterval(i time.Duration) *Validator[T] {
	v.sleepInterval = i
	return v

}

func (v *Validator[T]) Utime(utime int64) *Validator[T] {
	v.utime = utime
	return v
}

func (v *Validator[T]) Incr() *Validator[T] {
	v.fromBase = v.intrFromBase
	return v
}

func (v *Validator[T]) Validate(ctx context.Context) error {
	var eg errgroup.Group
	eg.Go(func() error {
		v.validateBaseToTarget(ctx)
		return nil
	})

	eg.Go(func() error {
		v.validateTargetToBase(ctx)
		return nil
	})
	return eg.Wait()
}

// <utime, id> 然后执行 SELECT * FROM xx WHERE utime > ? ORDER BY id
// 索引排序，还是内存排序？

// Validate 调用者可以通过 ctx 来控制校验程序退出
// 全量校验，是不是一条条比对？
// 所以要从数据库里面一条条查询出来
// utime 上面至少要有一个索引，并且 utime 必须是第一列
// <utime, col1, col2>, <utime> 这种可以
// <col1, utime> 这种就不可以
func (v *Validator[T]) validateBaseToTarget(ctx context.Context) {
	offset := 0
	for {
		//
		if v.highLoad.Load() {
			// 挂起
		}

		// 找到了 base 中的数据
		// 例如 .Order("id DESC")，每次插入数据，就会导致你的 offset 不准了
		// 如果我的表没有 id 这个列怎么办？
		// 找一个类似的列，比如说 ctime (创建时间）
		// 作业。你改成批量，性能要好很多
		src, err := v.fromBase(ctx, offset)
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			// 超时或者被人取消了
			return
		case nil:
			// 你真的查到了数据
			// 要去 target 里面找对应的数据
			var dst T
			err = v.target.Where("id = ?", src.ID()).First(&dst).Error
			// 我在这里，怎么办？
			switch err {
			case context.Canceled, context.DeadlineExceeded:
				// 超时或者被人取消了
				return
			case nil:
				// 找到了。你要开始比较
				// 你怎么比较？
				// 能不能这么比？
				// 1. src == dst
				// 这是利用反射来比
				// 这个原则上是可以的。
				//if reflect.DeepEqual(src, dst) {
				//
				//}
				//var srcAny any = src
				//if c1, ok := srcAny.(interface {
				//	// 有没有自定义的比较逻辑
				//	CompareTo(c2 migrator.Entity) bool
				//}); ok {
				//	// 有，我就用它的
				//	if !c1.CompareTo(dst) {
				//
				//	}
				//} else {
				//	// 没有，我就用反射
				//	if !reflect.DeepEqual(src, dst) {
				//
				//	}
				//}
				if !src.CompareTo(dst) {
					// 不相等
					// 这时候，我要干嘛？上报给 Kafka，就是告知数据不一致
					v.notify(ctx, src.ID(),
						events.InconsistentEventTypeNEQ)
				}

			case gorm.ErrRecordNotFound:
				// 这意味着，target 里面少了数据
				v.notify(ctx, src.ID(),
					events.InconsistentEventTypeTargetMissing)
			default:
				// 这里，要不要汇报，数据不一致？
				// 你有两种做法：
				// 1. 我认为，大概率数据是一致的，我记录一下日志，下一条
				v.l.Error("查询 target 数据失败", logger.Error(err))
				// 2. 我认为，出于保险起见，我应该报数据不一致，试着去修一下
				// 如果真的不一致了，没事，修它
				// 如果假的不一致（也就是数据一致），也没事，就是多余修了一次
				// 不好用哪个 InconsistentType
			}

		case gorm.ErrRecordNotFound:
			// 比完了。没数据了，全量校验结束了
			// 同时支持全量校验和增量校验，你这里就不能直接返回
			// 在这里，你要考虑：有些情况下，用户希望退出，有些情况下。用户希望继续
			// 当用户希望继续的时候，你要 sleep 一下
			if v.sleepInterval <= 0 {
				return
			}
			time.Sleep(v.sleepInterval)
			continue
		default:
			// 数据库错误
			v.l.Error("校验数据，查询 base 出错",
				logger.Error(err))
			// 课堂演示方便，你可以删掉
			time.Sleep(time.Second)
			// offset 最好是挪一下
			// 这里要不要挪
		}
		offset++
	}
}

func (v *Validator[T]) fullFromBase(ctx context.Context, offset int) (T, error) {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var src T
	// 找到了 base 中的数据
	// 例如 .Order("id DESC")，每次插入数据，就会导致你的 offset 不准了
	// 如果我的表没有 id 这个列怎么办？
	// 找一个类似的列，比如说 ctime (创建时间）
	// 作业。你改成批量，性能要好很多
	err := v.base.WithContext(dbCtx).
		// 最好不要取等号
		Offset(offset).
		Order("id").First(&src).Error
	return src, err
}

func (v *Validator[T]) intrFromBase(ctx context.Context, offset int) (T, error) {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var src T
	// 找到了 base 中的数据
	// 例如 .Order("id DESC")，每次插入数据，就会导致你的 offset 不准了
	// 如果我的表没有 id 这个列怎么办？
	// 找一个类似的列，比如说 ctime (创建时间）
	// 作业。你改成批量，性能要好很多
	err := v.base.WithContext(dbCtx).
		// 最好不要取等号
		Where("utime > ?", v.utime).
		Offset(offset).
		Order("utime ASC, id ASC").First(&src).Error

	// 等我琢磨一下
	// 按段取
	// WHERE utime >= ? LIMIT 10 ORDER BY UTIME
	// v.utime = srcList[len(srcList)].Utime()

	return src, err
}

// 通用写法，摆脱对 T 的依赖
//func (v *Validator[T]) intrFromBaseV1(ctx context.Context, offset int) (T, error) {
//	rows, err := v.base.WithContext(dbCtx).
//		// 最好不要取等号
//		Where("utime > ?", v.utime).
//		Offset(offset).
//		Order("utime ASC, id ASC").Rows()
//	cols, err := rows.Columns()
//	// 所有列的值
//	vals := make([]any, len(cols))
//	rows.Scan(vals...)
//	return vals
//}

// 理论上来说，可以利用 count 来加速这个过程，
// 我举个例子，假如说你初始化目标表的数据是 昨天的 23:59:59 导出来的
// 那么你可以 COUNT(*) WHERE ctime < 今天的零点，count 如果相等，就说明没删除
// 这一步大多数情况下效果很好，尤其是那些软删除的。
// 如果 count 不一致，那么接下来，你理论上来说，还可以分段 count
// 比如说，我先 count 第一个月的数据，一旦有数据删除了，你还得一条条查出来
// A utime=昨天
// A 在 base 里面，今天删了，A 在 target 里面，utime 还是昨天
// 这个地方，可以考虑不用 utime
// A 在删除之前，已经被修改过了，那么 A 在 target 里面的 utime 就是今天了
func (v *Validator[T]) validateTargetToBase(ctx context.Context) {
	// 先找 target，再找 base，找出 base 中已经被删除的
	// 理论上来说，就是 target 里面一条条找
	offset := 0
	for {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)

		var dstTs []T
		err := v.target.WithContext(dbCtx).
			Where("utime > ?", v.utime).
			Select("id").
			// WHERE 条件二分查找 COUNT
			Offset(offset).Limit(v.batchSize).
			Order("utime").Find(&dstTs).Error
		cancel()
		if len(dstTs) == 0 {
			// 没数据了。直接返回
			if v.sleepInterval <= 0 {
				return
			}
			time.Sleep(v.sleepInterval)
			continue
		}
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			// 超时或者被人取消了
			return
		// 正常来说，gorm 在 Find 方法接收的是切片的时候，不会返回 gorm.ErrRecordNotFound
		case gorm.ErrRecordNotFound:
			// 没数据了。直接返回
			if v.sleepInterval <= 0 {
				return
			}
			time.Sleep(v.sleepInterval)
			continue
		case nil:
			ids := slice.Map(dstTs, func(idx int, t T) int64 {
				return t.ID()
			})
			// 可以直接用 NOT IN
			var srcTs []T
			err = v.base.Where("id IN ?", ids).Find(&srcTs).Error
			switch err {
			case context.Canceled, context.DeadlineExceeded:
				// 超时或者被人取消了
				return
			case gorm.ErrRecordNotFound:
				v.notifyBaseMissing(ctx, ids)
			case nil:
				srcIds := slice.Map(srcTs, func(idx int, t T) int64 {
					return t.ID()
				})
				// 计算差集
				// 也就是，src 里面的咩有的
				diff := slice.DiffSet(ids, srcIds)
				v.notifyBaseMissing(ctx, diff)
			// 全没有
			default:
				// 记录日志
			}
		default:
			// 记录日志，continue 掉
			v.l.Error("查询target 失败", logger.Error(err))
		}
		offset += len(dstTs)
		if len(dstTs) < v.batchSize {
			if v.sleepInterval <= 0 {
				return
			}
			time.Sleep(v.sleepInterval)
		}
	}
}

func (v *Validator[T]) notifyBaseMissing(ctx context.Context, ids []int64) {
	for _, id := range ids {
		v.notify(ctx, id, events.InconsistentEventTypeBaseMissing)
	}
}

func (v *Validator[T]) notify(ctx context.Context, id int64, typ string) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	err := v.p.ProduceInconsistentEvent(ctx,
		events.InconsistentEvent{
			ID:        id,
			Direction: v.direction,
			Type:      typ,
		})
	cancel()
	if err != nil {
		// 这又是一个问题
		// 怎么办？
		// 你可以重试，但是重试也会失败，记日志，告警，手动去修
		// 我直接忽略，下一轮修复和校验又会找出来
		v.l.Error("发送数据不一致的消息失败", logger.Error(err))
	}
}
