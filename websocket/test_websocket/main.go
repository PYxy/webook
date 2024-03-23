package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

func main() {
	{
		upgrader := &websocket.Upgrader{
			//设置读写buffer
			//WriteBufferSize: 10,
			//ReadBufferSize:  20,
		} //测试内容
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
						fmt.Println("服务端读取数据失败：", err)
						_ = conn.Close()
						return
					}
					switch typ {
					case websocket.CloseMessage:
						fmt.Println("客户端主动结束:", err)
						_ = conn.Close()
						return
					default:
						fmt.Println("回写成功")
						if err = conn.WriteMessage(websocket.TextMessage, msg); err != nil {
							fmt.Println("服务端响应失败:", err)
							_ = conn.Close()
							return
						}
					}
				}
			}()
			//go func() {
			//	// 循环写一些消息到前端
			//	ticker := time.NewTicker(time.Second * 3)
			//	for now := range ticker.C {
			//		err := conn.WriteString("hello, " + now.String())
			//		if err != nil {
			//			// 也是连接崩了
			//			return
			//		}
			//	}
			//}()
		})
		http.ListenAndServe(":8081", nil)
	}
}

type Ws struct {
	*websocket.Conn
}

func (ws *Ws) WriteString(data string) error {
	err := ws.WriteMessage(websocket.TextMessage, []byte(data))
	return err
}
