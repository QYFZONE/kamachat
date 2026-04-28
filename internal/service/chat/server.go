package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"kama_chat_server/internal/dto/request"
	"kama_chat_server/internal/dto/respond"
	"kama_chat_server/internal/model"
	myredis "kama_chat_server/internal/service/redis"
	"kama_chat_server/pkg/constants"
	"kama_chat_server/pkg/enum/message/message_status_enum"
	"kama_chat_server/pkg/enum/message/message_type_enum"
	"kama_chat_server/pkg/util/random"
	"kama_chat_server/pkg/zlog"
)

// server.go 负责“所有在线连接的统一调度”。
//
// 一个最小的消息链路是：
// 前端 websocket -> Client.Read -> ChatServer.Transmit
// -> Server.handleTransmit -> 分发/落库/缓存
// -> 目标 Client.SendBack -> Client.Write -> 前端 websocket

type Server struct {
	// Clients 保存当前在线用户。
	// key 是用户 UUID，value 是对应的 websocket 连接包装对象。
	Clients map[string]*Client
	mutex   sync.RWMutex
	// Transmit 是所有客户端消息汇入服务端的总线。
	Transmit chan []byte
	// Login / Logout 用来串行处理客户端上下线事件。
	Login     chan *Client
	Logout    chan *Client
	closeOnce sync.Once
}

// ChatServer 是整个进程内共享的聊天服务单例。
var ChatServer *Server

// init 在包加载时初始化 ChatServer 和它的三条核心通道。
func init() {
	if ChatServer == nil {
		ChatServer = &Server{
			Clients:  make(map[string]*Client),
			Transmit: make(chan []byte, constants.CHANNEL_SIZE),
			Login:    make(chan *Client, constants.CHANNEL_SIZE),
			Logout:   make(chan *Client, constants.CHANNEL_SIZE),
		}
	}
}

// normalizePath 将带 host 的静态资源地址收敛为 /static/...，便于统一落库。
func normalizePath(path string) string {
	if path == "" || path == "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" {
		return path
	}
	staticIndex := strings.Index(path, "/static/")
	if staticIndex < 0 {
		zlog.Error("invalid static path: " + path)
		return path
	}
	return path[staticIndex:]
}

func isUserTarget(id string) bool {
	return strings.HasPrefix(id, "U")
}

// 约定：群聊 ID 以 G 开头。
func isGroupTarget(id string) bool {
	return strings.HasPrefix(id, "G")
}

// dispatchDirectMessage 把同一条单聊消息同时发给接收方和发送方。
// 这样发送方自己的聊天窗口也能立即收到一份回显。
func (s *Server) dispatchDirectMessage(senderID, receiverID string, payload any, messageUUID string) {
	s.sendJSONToClient(receiverID, payload, messageUUID)
	s.sendJSONToClient(senderID, payload, messageUUID)
}

// 群聊转发时只向当前在线的成员推送；离线成员靠历史消息补拉。
func (s *Server) dispatchGroupMessage(groupID string, payload any, messageUUID string) {
	members, err := chatStore.GetGroupMembers(groupID)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	for _, member := range members {
		s.sendJSONToClient(member, payload, messageUUID)
	}
}

// appendDirectCache 只在“缓存已经存在”的前提下做追加。
// 如果 Redis 没命中，这里不会新建缓存，而是交给后续历史消息查询回填。
func (s *Server) appendDirectCache(userOneID, userTwoID string, payload respond.GetMessageListRespond) {
	cacheKey := directMessageCacheKey(userOneID, userTwoID)
	cacheValue, err := myredis.GetKeyNilIsErr(cacheKey)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			zlog.Error(err.Error())
		}
		// 当前实现只在缓存已存在时做追加；首次未命中时交给后续历史查询回填。
		return
	}
	var items []respond.GetMessageListRespond
	if err := json.Unmarshal([]byte(cacheValue), &items); err != nil {
		zlog.Error(err.Error())
		return
	}

	items = append(items, payload)
	s.writeCache(cacheKey, items)
}

// appendGroupCache 和 appendDirectCache 的策略一致，只是 key 和响应类型不同。
func (s *Server) appendGroupCache(groupID string, payload respond.GetGroupMessageListRespond) {
	cacheKey := groupMessageCacheKey(groupID)
	cacheValue, err := myredis.GetKeyNilIsErr(cacheKey)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			zlog.Error(err.Error())
		}
		// 当前实现只在缓存已存在时做追加；首次未命中时交给后续历史查询回填。
		return
	}

	var items []respond.GetGroupMessageListRespond
	if err := json.Unmarshal([]byte(cacheValue), &items); err != nil {
		zlog.Error(err.Error())
		return
	}

	items = append(items, payload)
	s.writeCache(cacheKey, items)
}

// sendJSONToClient 是“发给某个在线用户”的最小单元。
// 它不会直接写 websocket，而是把 JSON 投递到目标连接的 SendBack 队列。
func (s *Server) sendJSONToClient(clientID string, payload any, messageUUID string) {
	client := s.GetClient(clientID)
	if client == nil {
		// 用户不在线时，这里静默返回，消息是否可补拉由历史消息逻辑决定。
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	select {
	case client.SendBack <- &MessageBack{Message: raw, Uuid: messageUUID}:
	default:
		zlog.Error("client send buffer is full: " + clientID)
	}
}

// handleLogin 把一个刚建好的 Client 放入在线表。
func (s *Server) handleLogin(client *Client) {
	if client == nil || client.Uuid == "" {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Clients[client.Uuid] = client
}

// handleLogout 只负责把连接从在线表移除。
// 真正的 socket 关闭由 client.shutdown 处理。
func (s *Server) handleLogout(client *Client) {
	if client == nil {
		return
	}
	s.RemoveClient(client.Uuid)
}

// shouldPersistAVEvent 定义“哪些音视频信令要入库”。
// 当前只持久化通话发起、接听、拒绝这几类关键状态变化。
func shouldPersistAVEvent(avData request.AVData) bool {
	if avData.MessageId != "PROXY" {
		return false
	}

	switch avData.Type {
	case "start_call", "receive_call", "reject_call":
		return true
	default:
		return false
	}
}

// newChatModelMessage 将 websocket 请求转换为普通消息模型。
func newChatModelMessage(req request.ChatMessageRequest) *model.Message {
	message := &model.Message{
		Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
		SessionId:  req.SessionId,
		Type:       req.Type,
		Content:    req.Content,
		Url:        req.Url,
		SendId:     req.SendId,
		SendName:   req.SendName,
		SendAvatar: normalizePath(req.SendAvatar),
		ReceiveId:  req.ReceiveId,
		FileType:   req.FileType,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		Status:     message_status_enum.Unsent,
		CreatedAt:  time.Now(),
		AVdata:     "",
	}

	if req.Type == message_type_enum.Text {
		// 文本消息不应该携带文件字段；这里统一清空，避免脏数据落库。
		message.Url = ""
		if message.FileSize == "" {
			message.FileSize = "0B"
		}
		message.FileType = ""
		message.FileName = ""
	}

	return message
}

// 音视频信令不复用文本/文件字段，只保留 AVdata。
func newAVModelMessage(req request.ChatMessageRequest) *model.Message {
	return &model.Message{
		Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
		SessionId:  req.SessionId,
		Type:       req.Type,
		Content:    "",
		Url:        "",
		SendId:     req.SendId,
		SendName:   req.SendName,
		SendAvatar: normalizePath(req.SendAvatar),
		ReceiveId:  req.ReceiveId,
		FileType:   "",
		FileName:   "",
		FileSize:   "",
		Status:     message_status_enum.Unsent,
		CreatedAt:  time.Now(),
		AVdata:     req.AVdata,
	}
}

// handleChatMessage 处理普通文本/文件消息。
//
// 顺序是：
// 1. 先把请求转成消息模型并落库。
// 2. 按接收方类型决定是单聊还是群聊。
// 3. 组装回包结构。
// 4. 推送给在线用户。
// 5. 尝试追加 Redis 缓存。
func (s *Server) handleChatMessage(req request.ChatMessageRequest) {
	message := newChatModelMessage(req)
	if err := chatStore.SaveMessage(message); err != nil {
		zlog.Error(err.Error())
		return
	}

	if isUserTarget(req.ReceiveId) {
		// 单聊响应结构。
		payload := respond.GetMessageListRespond{
			SendId:     message.SendId,
			SendName:   message.SendName,
			SendAvatar: message.SendAvatar,
			ReceiveId:  message.ReceiveId,
			Content:    message.Content,
			Url:        message.Url,
			Type:       message.Type,
			FileType:   message.FileType,
			FileName:   message.FileName,
			FileSize:   message.FileSize,
			CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		// 单聊既要发给接收方，也要回显给发送方自己。
		s.dispatchDirectMessage(req.SendId, req.ReceiveId, payload, message.Uuid)
		s.appendDirectCache(req.SendId, req.ReceiveId, payload)
		return
	}

	if isGroupTarget(req.ReceiveId) {
		// 群聊响应结构。
		payload := respond.GetGroupMessageListRespond{
			SendId:     message.SendId,
			SendName:   message.SendName,
			SendAvatar: message.SendAvatar,
			ReceiveId:  message.ReceiveId,
			Content:    message.Content,
			Url:        message.Url,
			Type:       message.Type,
			FileType:   message.FileType,
			FileName:   message.FileName,
			FileSize:   message.FileSize,
			CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		s.dispatchGroupMessage(req.ReceiveId, payload, message.Uuid)
		s.appendGroupCache(req.ReceiveId, payload)
		return
	}

	zlog.Error("unsupported receive id: " + req.ReceiveId)
}

// handleAVMessage 处理音视频信令消息。
//
// 和普通聊天消息的区别有两点：
// 1. 音视频业务数据在 req.AVdata 里，需要二次反序列化。
// 2. 不是所有信令都持久化，只有 shouldPersistAVEvent 判定的关键事件才入库。
func (s *Server) handleAVMessage(req request.ChatMessageRequest) {
	var avData request.AVData
	if err := json.Unmarshal([]byte(req.AVdata), &avData); err != nil {
		zlog.Error(err.Error())
		return
	}

	message := newAVModelMessage(req)
	if shouldPersistAVEvent(avData) {
		if err := chatStore.SaveMessage(message); err != nil {
			zlog.Error(err.Error())
			return
		}
	}

	if !isUserTarget(req.ReceiveId) {
		return
	}

	payload := respond.AVMessageRespond{
		SendId:     message.SendId,
		SendName:   message.SendName,
		SendAvatar: message.SendAvatar,
		ReceiveId:  message.ReceiveId,
		Type:       message.Type,
		Content:    message.Content,
		Url:        message.Url,
		FileType:   message.FileType,
		FileName:   message.FileName,
		FileSize:   message.FileSize,
		CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
		AVdata:     message.AVdata,
	}

	messageUUID := ""
	if shouldPersistAVEvent(avData) {
		messageUUID = message.Uuid
	}

	// 当前实现只把音视频信令发给对端，不给发送方再做一份回显。
	s.sendJSONToClient(req.ReceiveId, payload, messageUUID)
}

// handleTransmit 是聊天服务处理一条消息帧的统一入口。
// 这里只负责“解包 + 基本字段校验 + 按类型分发”。
func (s *Server) handleTransmit(data []byte) {
	var req request.ChatMessageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		zlog.Error(err.Error())
		return
	}

	if req.SendId == "" || req.ReceiveId == "" {
		zlog.Error("invalid chat message: empty sender or receiver")
		return
	}

	switch req.Type {
	case message_type_enum.Text, message_type_enum.File:
		s.handleChatMessage(req)
	case message_type_enum.AudioOrVideo:
		s.handleAVMessage(req)
	default:
		zlog.Error(fmt.Sprintf("unsupported message type: %d", req.Type))
	}
}

// Start 是 ChatServer 的主事件循环。
// 只有它启动后，Login / Logout / Transmit 三条通道才会真正被消费。
func (s *Server) Start() {
	for {
		select {
		case client, ok := <-s.Login:
			if !ok {
				return
			}
			s.handleLogin(client)
		case client, ok := <-s.Logout:
			if !ok {
				return
			}
			s.handleLogout(client)
		case data, ok := <-s.Transmit:
			if !ok {
				return
			}
			s.handleTransmit(data)
		}
	}
}

// GetClient 读取当前在线表。
func (s *Server) GetClient(uuid string) *Client {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Clients[uuid]
}

// directMessageCacheKey 把双方用户 ID 排序后再拼接，
// 避免 A:B 和 B:A 形成两份单聊缓存。
func directMessageCacheKey(userOneID, userTwoID string) string {
	left, right := userOneID, userTwoID
	if left > right {
		left, right = right, left
	}
	return "message:list:" + left + ":" + right
}

func groupMessageCacheKey(groupID string) string {
	return "message:group_list:" + groupID
}

// writeCache 统一负责把消息列表编码并写入 Redis。
func (s *Server) writeCache(cacheKey string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	if err := myredis.SetKeyEx(cacheKey, string(raw), time.Minute*constants.REDIS_TIMEOUT); err != nil {
		zlog.Error(err.Error())
	}
}

// RemoveClient 从在线表中移除一个用户。
func (s *Server) RemoveClient(uuid string) {
	s.mutex.Lock()
	delete(s.Clients, uuid)
	s.mutex.Unlock()
}

// SendClientToLogin 非阻塞地投递登录事件。
func (s *Server) SendClientToLogin(client *Client) {
	if client == nil {
		return
	}

	select {
	case s.Login <- client:
	default:
		zlog.Error("login channel is full")
	}
}

// SendClientToLogout 非阻塞地投递登出事件。
func (s *Server) SendClientToLogout(client *Client) {
	if client == nil {
		return
	}
	select {
	case s.Logout <- client:
	default:
		zlog.Error("logout channel is full")
	}
}
