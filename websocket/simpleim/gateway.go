package simpleim

import (
	"context"
	"encoding/json"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"gitee.com/geekbang/basic-go/webook/pkg/saramax"
	"github.com/IBM/sarama"
	"github.com/ecodeclub/ekit/syncx"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"time"
)

type WsGateway struct {
	// 连接了这个实例的客户端
	// 这里我们用 uid 作为 key
	// 实践中要考虑到不同的设备，
	// 那么这个 key 可能是一个复合结构，例如 uid + 设备
	conns *syncx.Map[int64, *Conn]
	svc   *IMService

	client     sarama.Client
	instanceId string
	upgrader   *websocket.Upgrader
}

// Start 在这个启动的时候，监听 websocket 的请求，然后转发到后端
func (g *WsGateway) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", g.wsHandler)
	err := g.subscribeMsg()
	if err != nil {
		return err
	}
	return http.ListenAndServe(addr, mux)
}

// MessageV1 和前端约定好，具体的消息的内容的格式
//type MessageV1 struct {
//
//	// 这个是前端的序列号
//	// 不要求全局唯一的，正常只要当下这个 websocket 唯一就可以
//	Seq string
//
//	// 谁发的？
//	// 能不能是前端传过来的？
//	// Sender int64
//
//	// 发给谁
//	// cid channel id(group id)，聊天 ID
//	// 单聊，也是用聊天 ID
//	Cid int64
//	// 内容
//	// Type 这个消息是什么消息
//	// 这个是你 IM 内部的类型
//	// type = "video", => content = url/资源标识符 key
//	// content 不可能是视频本身
//	// {"title": "GO从入门到入土", Addr: "https://oss.aliyun.com/im/resource/abc"}
// @某人 {"metions": []int64, "text": }
//	Type string
//	// 你有文本消息，你有图片消息，你有视频消息
//	// 你这个 Content 究竟是什么？
//	Content string
//
//	// 万一你每个消息都要校验 token，可以在这里带
//	//Token string
//}

func (g *WsGateway) subscribeMsg() error {
	// 用 instance id 作为消费者组
	// 不像业务里面，同样的节点同一个消费者组
	// 每个节点单独的消费者组
	cg, err := sarama.NewConsumerGroupFromClient(g.instanceId,
		g.client)
	if err != nil {
		return err
	}
	go func() {
		err := cg.Consume(context.Background(),
			[]string{eventName},
			saramax.NewHandler[Event](logger.NewNoOpLogger(), g.consume))
		if err != nil {
			log.Println("退出监听消息循环", err)
		}
	}()
	return nil
}

func (g *WsGateway) wsHandler(writer http.ResponseWriter, request *http.Request) {
	conn, err := g.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		// 升级失败
		writer.Write([]byte("升级 ws 失败"))
		return
	}

	// 在这里拿到 session。
	// 如果我在这里拿到了 session
	// 模拟我从 session/token 里面拿到 uid
	c := &Conn{
		Conn: conn,
	}
	uid := g.Uid(request)
	// 我记录一下，哪些人连上了我
	g.conns.Store(uid, c)
	// 就是我得拿到你的 session
	go func() {
		defer func() {
			g.conns.Delete(uid)
		}()
		for {
			// 在这里监听用户发过来的消息
			// typ 一般不需要处理，前端和你会约定好，typ 是什么
			// websocket 这里你拿不到 token
			typ, msgBytes, err := c.ReadMessage()
			//switch err {
			//case context.DeadlineExceeded:
			//	// 这个地方你是可以继续的
			//	continue
			//case nil:
			//
			//default:
			//	// 都是网络出了问题，或者你的连接出了任务
			//	return
			//}
			if err != nil {
				return
			}

			switch typ {
			case websocket.TextMessage, websocket.BinaryMessage:
				// 你是不是得知道，谁发的？发给谁？内容是什么？

				var msg Message
				err = json.Unmarshal(msgBytes, &msg)
				if err != nil {
					// 格式不对，正常不可能进来
					continue
				}

				go func() {
					// 我是建议开的
					// 开 goroutine 的危险
					// 搞协程池（任务池），控制住 goroutine 的数量
					// 再开一个 goroutine
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
					err = g.svc.Receive(ctx, uid, msg)
					cancel()
					if err != nil {
						// 引入重试
						// 你是不是要告诉前端，你出错了
						// 前端怎么知道我哪条出错了？
						err = c.Send(Message{
							Seq:     msg.Seq,
							Type:    "result",
							Content: "failed",
						})
						if err != nil {
							// 记录日志
							// 这里也可以引入重试
						}
					}
				}()

			case websocket.CloseMessage:
				c.Close()
			default:

			}
		}
	}()
}

// Uid 一般是从 jwt token 或者 session 里面取出来
// 这里模拟从 header 里面读取出来
func (g *WsGateway) Uid(req *http.Request) int64 {

	// 拿到 token
	//token := strings.TrimLeft(req.Header.Get("Authorization"), "Bearer ")
	// jwt 解析
	// jwt.Parse
	// req.Cookie("sess_id")

	uidStr := req.Header.Get("uid")
	uid, _ := strconv.ParseInt(uidStr, 10, 64)
	return uid
}

func (g *WsGateway) consume(msg *sarama.ConsumerMessage, evt Event) error {
	// 转发
	// 我怎么知道，这个 receiver 有没有连上我？
	// 多端同步的时候，还需要知道哪个设备连上了我
	receiverConn, ok := g.conns.Load(evt.Receiver)
	if !ok {
		return nil
	}
	return receiverConn.Send(evt.Msg)
}

// Conn 稍微做一个封装
type Conn struct {
	*websocket.Conn
}

func (c *Conn) Send(msg Message) error {
	val, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, val)
}

type Message struct {
	// 发过来的消息的序列号
	// 用于前后端关联消息
	Seq string
	// 这个是后端的 ID
	// 前端有时候支持引用功能，转发功能的时候，会需要这个 ID
	ID int64
	// 用来标识不同的消息类型
	// 文本消息，视频消息
	// 系统消息（后端往前端发的，跟 IM 本身管理有关的消息）
	Type    string
	Content string
	// 聊天 ID，注意，正常来说这里不是记录目标用户 ID
	// 而是记录代表了这个聊天的 ID
	Cid int64
}
