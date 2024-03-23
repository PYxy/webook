package simpleim

type Event struct {
	Msg Message
	// 接收者
	Receiver int64
	// 发送的 device
	Device string
}

// EventV1 扩散只会和你有多少接入节点有关
// 和群里面有多少人无关
// 注册与发现机制，那么你就可以精确控制，转发到哪些节点
type EventV1 struct {
	Msg       Message
	Receivers []int64
}

const eventName = "simple_im_msg"
