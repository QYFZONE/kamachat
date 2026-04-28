# Chat Module Flow

这份文档只聚焦 `internal/service/chat` 这一块，方便后面继续改 websocket、消息分发、群聊和音视频逻辑时快速定位。

## 1. 模块职责总览

- `client.go`：管理“一个 websocket 连接”的收和发。
- `server.go`：管理“所有在线连接”的登录、登出、消息分发、落库和缓存。
- `store.go`：给 `chat` 包提供最小存储能力，隔离底层 DAO 细节。

## 2. 整体架构图

```mermaid
flowchart LR
    FE[前端 WebSocket] --> WSLOGIN[api/v1 WsLogin]
    WSLOGIN --> INIT[chat.NewClientInit]
    INIT --> CLIENT[Client 实例]
    CLIENT --> READ[Client.Read]
    CLIENT --> WRITE[Client.Write]

    READ --> TRANSMIT[ChatServer.Transmit]
    INIT --> LOGIN[ChatServer.Login]
    LOGOUTAPI[api/v1 WsLogout] --> LOGOUT[ChatServer.Logout]

    LOGIN --> START[ChatServer.Start 主循环]
    LOGOUT --> START
    TRANSMIT --> START

    START --> HANDLELOGIN[handleLogin]
    START --> HANDLELOGOUT[handleLogout]
    START --> HANDLETX[handleTransmit]

    HANDLETX --> CHATMSG[handleChatMessage]
    HANDLETX --> AVMSG[handleAVMessage]

    CHATMSG --> STORE[chatStore]
    AVMSG --> STORE
    STORE --> DAO[dao.Message / dao.Group]
    CHATMSG --> REDIS[Redis 缓存]

    CHATMSG --> SENDBACK[目标 Client.SendBack]
    AVMSG --> SENDBACK
    SENDBACK --> WRITE
    WRITE --> FE
```

## 3. 连接建立与销毁

```mermaid
flowchart TD
    A[前端发起 websocket 连接] --> B[WsLogin 读取 client_id]
    B --> C[NewClientInit]
    C --> D[HTTP 升级为 websocket]
    D --> E[创建 Client]
    E --> F[创建 SendTo 和 SendBack 缓冲队列]
    F --> G[投递到 ChatServer.Login]
    G --> H[启动 Client.Read goroutine]
    G --> I[启动 Client.Write goroutine]
    G --> J[ChatServer.Start 消费 Login]
    J --> K[handleLogin 写入在线表 Clients]

    L[前端主动登出或连接异常] --> M[ClientLogout / shutdown]
    M --> N[投递到 ChatServer.Logout]
    M --> O[关闭 websocket]
    M --> P[关闭 SendTo / SendBack]
    N --> Q[ChatServer.Start 消费 Logout]
    Q --> R[handleLogout / RemoveClient]
```

## 4. 普通消息链路：单聊与群聊

### 4.1 单聊消息

```mermaid
sequenceDiagram
    participant FE1 as 发送方前端
    participant C1 as 发送方 Client.Read
    participant S as ChatServer.Start
    participant H as handleChatMessage
    participant ST as chatStore
    participant DB as MySQL
    participant C2 as 接收方 Client.Write
    participant C1W as 发送方 Client.Write
    participant FE2 as 接收方前端

    FE1->>C1: websocket 发送 ChatMessageRequest
    C1->>C1: validateMessage
    C1->>C1: flushPendingMessages
    C1->>S: 投递到 ChatServer.Transmit
    S->>H: handleTransmit -> handleChatMessage
    H->>H: newChatModelMessage
    H->>ST: SaveMessage
    ST->>DB: dao.Message.CreateMessage
    H->>C2: sendJSONToClient(接收方)
    H->>C1W: sendJSONToClient(发送方自己)
    C2->>FE2: 写回 websocket
    C1W->>FE1: 写回 websocket 回显
    C2->>ST: UpdateMessageStatus(Sent)
    C1W->>ST: UpdateMessageStatus(Sent)
```

### 4.2 群聊消息

```mermaid
sequenceDiagram
    participant FE as 发送方前端
    participant C as 发送方 Client.Read
    participant S as ChatServer.Start
    participant H as handleChatMessage
    participant ST as chatStore
    participant DB as MySQL
    participant GDAO as dao.Group
    participant M1 as 成员1 Client.Write
    participant M2 as 成员2 Client.Write
    participant M3 as 成员3 Client.Write

    FE->>C: websocket 发送群聊消息
    C->>S: 投递到 ChatServer.Transmit
    S->>H: handleTransmit -> handleChatMessage
    H->>ST: SaveMessage
    ST->>DB: dao.Message.CreateMessage
    H->>ST: GetGroupMembers(groupID)
    ST->>GDAO: GetGroupInfoByGroupId
    GDAO-->>ST: group.Members JSON
    ST-->>H: []string 成员列表
    H->>M1: sendJSONToClient
    H->>M2: sendJSONToClient
    H->>M3: sendJSONToClient
```

## 5. 音视频信令链路

```mermaid
flowchart TD
    A[前端发送 AudioOrVideo 消息] --> B[Client.Read]
    B --> C[ChatServer.Transmit]
    C --> D[handleTransmit]
    D --> E[handleAVMessage]
    E --> F[解析 req.AVdata 为 AVData]
    F --> G{shouldPersistAVEvent?}
    G -- 是 --> H[newAVModelMessage]
    H --> I[chatStore.SaveMessage]
    G -- 否 --> J[跳过落库]
    I --> K[sendJSONToClient(仅接收方)]
    J --> K
    K --> L[接收方 Client.Write]
    L --> M[接收方前端]
```

## 6. Client 内部两级缓冲

这部分最容易读乱，单独画出来。

```mermaid
flowchart TD
    A[Read 从 websocket 收到新消息] --> B[validateMessage]
    B --> C[flushPendingMessages]
    C --> D{ChatServer.Transmit 可写?}
    D -- 是 --> E[旧消息先发到服务端]
    D -- 否 --> F[旧消息退回 SendTo]
    F --> G[停止冲刷]
    E --> H[继续处理当前新消息]
    G --> H
    H --> I{当前消息可直接写入 Transmit?}
    I -- 是 --> J[交给 ChatServer 处理]
    I -- 否 --> K{当前消息可写入 SendTo?}
    K -- 是 --> L[暂存到当前连接本地缓冲]
    K -- 否 --> M[回前端 too many pending messages]
```

## 7. 在线表与消息总线

```mermaid
flowchart LR
    subgraph Client Side
        A1[Client.Read]
        A2[Client.Write]
        A3[SendTo 本地上行缓冲]
        A4[SendBack 本地下行缓冲]
    end

    subgraph Server Side
        B1[Login]
        B2[Logout]
        B3[Transmit]
        B4[Start 主循环]
        B5[Clients 在线表]
    end

    A1 --> B3
    A1 --> A3
    B1 --> B4
    B2 --> B4
    B3 --> B4
    B4 --> B5
    B4 --> A4
    A4 --> A2
```

## 8. 当前实现里要特别记住的几个点

1. `Client.Read` 不做真正业务分发，它只负责把消息送进 `ChatServer.Transmit`。
2. `Client.Write` 才是真正写 websocket 的地方，所以消息状态更新也放在这里。
3. 单聊消息会回显给发送方自己，群聊消息会广播给所有在线成员。
4. `appendDirectCache` 和 `appendGroupCache` 只在缓存已存在时追加，不会在未命中时创建新缓存。
5. 音视频消息不是全部落库，只有 `shouldPersistAVEvent` 判定的关键事件才持久化。

## 9. 运行前提

当前代码结构要求 `ChatServer.Start()` 真的跑起来，`Login`、`Logout`、`Transmit` 三条通道才会被消费。

你现在的 [cmd/kama_chat_server/main.go](../cmd/kama_chat_server/main.go) 里，这段逻辑还是注释状态：

```go
// if kafkaConfig.MessageMode == "channel" {
//     go chat.ChatServer.Start()
// } else {
//     go chat.KafkaChatServer.Start()
// }
```

如果你后面要继续验证这套 channel 版 chat 流程，这里需要恢复启动。
