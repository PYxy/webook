package events

type InconsistentEvent struct {
	ID int64
	// 用什么来修，取值为 SRC，意味着，以源表为准，取值为 DST，以目标表为准
	Direction string
	// 有些时候，一些观测，或者一些第三方，需要知道，是什么引起的不一致
	// 因为他要去 DEBUG
	// 这个是可选的
	Type string

	// 事件里面带 base 的数据
	// 修复数据用这里的去修，这种做法是不行的，因为有严重的并发问题
	Columns map[string]any
}

const (
	// InconsistentEventTypeTargetMissing 校验的目标数据，缺了这一条
	InconsistentEventTypeTargetMissing = "target_missing"
	// InconsistentEventTypeNEQ 不相等
	InconsistentEventTypeNEQ         = "neq"
	InconsistentEventTypeBaseMissing = "base_missing"
)

//type Fixer struct {
//	base   *gorm.DB
//	target *gorm.DB
//}
