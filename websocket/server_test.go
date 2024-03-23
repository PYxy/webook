package websocket

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	upgrader := &websocket.Upgrader{}
	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		// 这个就是用来搞升级的，或者说初始化 ws 的
		// conn 代表一个 websocket 连接
		c, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			// 升级失败
			writer.Write([]byte("升级 ws 失败"))
			return
		}
		conn := &Ws{Conn: c}
		// 从 websocket 接收数据
		go func() {
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
					conn.Close()
					return
				default:
					t.Log(typ, string(msg))
				}
			}
		}()
		go func() {
			// 循环写一些消息到前端
			ticker := time.NewTicker(time.Second * 3)
			for now := range ticker.C {
				err := conn.WriteString("hello, " + now.String())
				if err != nil {
					// 也是连接崩了
					return
				}
			}
		}()
	})
	go func() {
		server := gin.Default()
		server.GET("/", func(ctx *gin.Context) {
			// req := ctx.Request
			go func() {
				// 在这里继续使用 ctx，就可能被坑

			}()
			//gorm.DB{}
			//ctx.String()
		})
		//server.ServeHTTP()
		server.Run(":8082")
	}()
	http.ListenAndServe(":8081", nil)
}

type Ws struct {
	*websocket.Conn
}

func (ws *Ws) WriteString(data string) error {
	err := ws.WriteMessage(websocket.TextMessage, []byte(data))
	return err
}

//
//func Read() {
//	var conn net.Conn
//
//	for {
//		// 如果每一次都创建这个 buffer，
//		buffer := pool.Get()
//		conn.Read(buffer)
//		// 用完了
//		pool.Put(buffer)
//	}
//}
