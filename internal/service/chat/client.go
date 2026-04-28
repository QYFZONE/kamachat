package chat

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"kama_chat_server/internal/dto/request"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/enum/message/message_status_enum"
	"kama_chat_server/pkg/zlog"
)

// client.go 只处理“一个 websocket 连接”的生命周期。
//
// 这一层不做真正的消息分发，它负责两件事：
// 1. 从前端读消息，然后把消息投递给 ChatServer.Transmit。
// 2. 从自己的 SendBack 队列取消息，再写回前端 websocket。
//
// 可以把 Client 理解成“某个用户在当前机器上的一个在线连接”。

type MessageBack struct {
	// Message 是已经编码好的 JSON 响应体，最终会原样写回 websocket。
	Message []byte
	// Uuid 是消息表里的消息 UUID。
	// 为空表示这条回包只做透传，不需要回写“已发送”状态。
	Uuid    string
}

type Client struct {
	// Conn 是当前用户和服务端之间的真实 websocket 连接。
	Conn *websocket.Conn
	// Uuid 是当前连接所属的用户 ID。
	Uuid string
	// SendTo 是当前连接的本地上行缓冲；全局发送通道繁忙时先暂存到这里。
	SendTo chan []byte
	// SendBack 是发给当前连接的下行缓冲，由服务端分发逻辑写入。
	SendBack  chan *MessageBack
	closeOnce sync.Once
}

// upgrader 负责把 HTTP 请求升级为 websocket。
// 这里的 CheckOrigin 直接放行，等于不做来源校验。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Read 持续接收前端发来的 websocket 消息。
//
// 流程是：
// 1. 先从 socket 读一帧。
// 2. 校验这帧数据至少能反序列化成 ChatMessageRequest。
// 3. 先尝试冲刷这个连接之前积压的消息，尽量保持发送顺序。
// 4. 再把当前消息投递到 ChatServer.Transmit。
// 5. 如果总线繁忙，就退回到当前连接自己的 SendTo 本地缓冲。
func (c *Client) Read() {
	zlog.Info("ws read goroutine start")

	for {
		// payload 是前端发来的一条完整 websocket 消息。
		_, payload, err := c.Conn.ReadMessage()
		if err != nil {
			zlog.Error(err.Error())
			c.shutdown()
			return
		}

		if !c.validateMessage(payload) {
			continue
		}

		// 先发旧消息，再发新消息，避免后来的消息抢在前面出队。
		c.flushPendingMessages()

		// 优先直接进入服务端总线，让 ChatServer 统一处理。
		if c.trySendToServer(payload) {
			continue
		}

		// 总线满了就先落到当前连接自己的本地缓冲。
		if c.tryBufferMessage(payload) {
			continue
		}

		// 两级缓冲都满了，说明当前连接的待发送消息已经堆积过多。
		if err := c.Conn.WriteMessage(websocket.TextMessage, []byte("too many pending messages, please retry later")); err != nil {
			zlog.Error(err.Error())
			c.shutdown()
			return
		}
	}

}

// Write 持续把服务端分发给当前连接的消息写回 websocket。
//
// 注意这里的消息来源不是前端，而是 ChatServer.sendJSONToClient
// 往 c.SendBack 里投递的数据。
func (c *Client) Write() {
	zlog.Info("ws write goroutine start")
	for messageBack := range c.SendBack {
		// 只有真正写到 websocket 成功后，才算这条回包已经送达当前连接。
		if err := c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message); err != nil {
			zlog.Error(err.Error())
			c.shutdown()
			return
		}

		// 没有消息 UUID 的回包只负责透传，不需要更新持久化状态。
		if messageBack.Uuid == "" {
			continue
		}

		if err := chatStore.UpdateMessageStatus(messageBack.Uuid, message_status_enum.Sent); err != nil {
			zlog.Error(err.Error())
		}
	}
}

// validateMessage 只做最基础的格式校验：
// 能否把 payload 解析为约定的消息结构。
// 更细的业务校验会在 server.go 的 handleTransmit 里继续做。
func (c *Client) validateMessage(payload []byte) bool {
	var message request.ChatMessageRequest
	if err := json.Unmarshal(payload, &message); err != nil {
		zlog.Error(err.Error())
		return false
	}
	return true
}

// trySendToServer 非阻塞地把消息投递到服务端总线。
// 返回 true 代表本次投递成功；false 代表总线当前已满。
func (c *Client) trySendToServer(payload []byte) bool {
	select {
	case ChatServer.Transmit <- payload:
		return true
	default:
		return false
	}
}

// tryBufferMessage 非阻塞地把消息放进当前连接的本地缓冲。
// 这里只是“暂存”，真正消费它的人仍然是 flushPendingMessages。
func (c *Client) tryBufferMessage(payload []byte) bool {
	select {
	case c.SendTo <- payload:
		return true
	default:
		return false
	}
}

// flushPendingMessages 尝试把当前连接历史积压的消息重新送回服务端总线。
//
// 这里用 for + select(default) 的写法，是为了“尽可能多冲刷，
// 但又不在没有数据时阻塞当前 goroutine”。
func (c *Client) flushPendingMessages() {
	for {
		select {
		case pendingMessage, ok := <-c.SendTo:
			if !ok {
				return
			}
			// 全局通道仍然繁忙时，把当前消息重新放回本地缓冲并停止冲刷。
			// 这里一旦停止，后续会优先把本次 Read 里的当前消息也留在本地缓冲。
			if !c.trySendToServer(pendingMessage) {
				c.tryBufferMessage(pendingMessage)
				return
			}
		default:
			return
		}
	}
}

// shutdown 负责清理一个连接关联的所有本地资源。
// closeOnce 用来防止多条错误路径同时触发重复 close。
func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		ChatServer.RemoveClient(c.Uuid)
		_ = c.Conn.Close()
		close(c.SendTo)
		close(c.SendBack)
	})
}

// NewClientInit 是 websocket 建连入口。
//
// 它做的事情很固定：
// 1. 把 HTTP 升级成 websocket。
// 2. 创建 Client 对象和两个缓冲队列。
// 3. 把这个 Client 注册到 ChatServer。
// 4. 分别启动读协程和写协程。
func NewClientInit(c *gin.Context, clientId string) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	client := &Client{
		Conn:     conn,
		Uuid:     clientId,
		SendTo:   make(chan []byte, constants.CHANNEL_SIZE),
		SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE),
	}

	ChatServer.SendClientToLogin(client)
	go client.Read()
	go client.Write()

	zlog.Info("ws connection established")
}

// ClientLogout 主动断开指定用户的 websocket 连接。
// 如果该用户当前不在线，就直接视为登出成功。
func ClientLogout(clientId string) (string, int) {
	client := ChatServer.GetClient(clientId)
	if client == nil {
		return "logout success", 0
	}

	ChatServer.SendClientToLogout(client)
	client.shutdown()
	return "logout success", 0
}
