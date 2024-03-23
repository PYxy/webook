package websocket

import (
	"github.com/ecodeclub/ekit/syncx"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"testing"
)

// 集线器/中转站
type Hub struct {
	// syncx.Map 是我对 sync.Map 的一个简单封装
	// 连上了我这个节点的所有的 websocket 的连接
	// key 是客户端的名称
	// 绝大多数情况下，你得存着这个东西
	conns *syncx.Map[string, *websocket.Conn]

	// sync.Map
}

func (h *Hub) AddConn(name string, conn *websocket.Conn) {
	h.conns.Store(name, conn)
	go func() {
		// 准备接收数据
		for {
			// typ 是指 websocket 里面的消息类型
			typ, msg, err := conn.ReadMessage()
			// 这个 error 很难处理
			if err != nil {
				// 基本上这里都是代表连接出了问题
				return
			}
			switch typ {
			case websocket.CloseMessage:
				h.conns.Delete(name)
				conn.Close()
				return
			default:
				// 要转发了
				log.Println("来自客户端", name, typ, string(msg))
				h.conns.Range(func(key string, value *websocket.Conn) bool {
					if key == name {
						// 自己的，就不需要转发了
						return true
					}
					log.Println("转发给", key)
					err := value.WriteMessage(typ, msg)
					if err != nil {
						log.Println(err)
					}
					return true
				})
			}
		}
	}()
}

func TestHub(t *testing.T) {
	upgrader := &websocket.Upgrader{}
	hub := &Hub{conns: &syncx.Map[string, *websocket.Conn]{}}
	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		c, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			// 升级失败
			writer.Write([]byte("升级 ws 失败"))
			return
		}
		name := request.URL.Query().Get("name")
		hub.AddConn(name, c)
	})
	http.ListenAndServe(":8081", nil)
}
