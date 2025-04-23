package user

import (
	"chatroom-go/common"
	"chatroom-go/message"
	"chatroom-go/room"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var roomManager *room.RoomManager

// User 实现common.User接口
// 心跳相关常量
const (
	heartbeatInterval = 10 * time.Second // 心跳间隔
	heartbeatTimeout  = 3                // 心跳超时次数
)

type User struct {
	ID              string          // 用户唯一标识
	Name            string          // 用户名称
	Conn            *websocket.Conn // WebSocket连接
	roomID          string          // 当前所在房间ID
	msgChan         chan []byte     // 消息通道
	quitChan        chan struct{}   // 退出信号通道
	lastHeartbeat   time.Time       // 最后一次心跳时间
	heartbeatFailed int             // 心跳失败次数
	heartbeatMu     sync.Mutex      // 心跳相关的互斥锁
}

// UserManager 用户管理器
type UserManager struct {
	users sync.Map // 存储所有在线用户
	mu    sync.RWMutex
}

// NewUser 创建新用户
func NewUser(id, name string, conn *websocket.Conn) *User {
	return &User{
		ID:            id,
		Name:          name,
		Conn:          conn,
		msgChan:       make(chan []byte, 100),
		quitChan:      make(chan struct{}),
		lastHeartbeat: time.Now(),
	}
}

// GetID 获取用户ID
func (u *User) GetID() string {
	return u.ID
}

// GetName 获取用户名称
func (u *User) GetName() string {
	return u.Name
}

// GetRoomID 获取用户当前所在房间ID
func (u *User) GetRoomID() string {
	return u.roomID
}

// SetRoomID 设置用户当前所在房间ID
func (u *User) SetRoomID(roomID string) {
	u.roomID = roomID
}

// Start 启动用户的消息处理
func (u *User) Start() {
	go u.readPump()
	go u.writePump()
}

// Stop 停止用户的消息处理
func (u *User) Stop() {
	close(u.quitChan)
}

// SendMessage 发送消息给用户
func (u *User) SendMessage(msg []byte) {
	select {
	case u.msgChan <- msg:
	default:
		// 通道已满，消息丢弃
	}
}

// readPump 处理来自客户端的消息
func (u *User) readPump() {
	defer func() {
		u.Conn.Close()
	}()

	// 启动心跳检测
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			u.checkHeartbeat()
		}
	}()

	for {
		select {
		case <-u.quitChan:
			return
		default:
			_, message, err := u.Conn.ReadMessage()
			if err != nil {
				break
			}
			// 处理接收到的消息
			u.handleMessage(message)
		}
	}
}

// writePump 向客户端发送消息
func (u *User) writePump() {
	defer func() {
		u.Conn.Close()
	}()

	// 启动心跳发送
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case message := <-u.msgChan:
			err := u.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}
		case <-ticker.C:
			// 发送心跳消息
			heartbeatMsg := message.NewMessage(message.HeartbeatMessage, "", u.Name, u.roomID)
			msgData, err := heartbeatMsg.ToJSON()
			if err != nil {
				log.Printf("心跳消息序列化失败: %v\n", err)
				continue
			}
			err = u.Conn.WriteMessage(websocket.TextMessage, msgData)
			if err != nil {
				return
			}
		case <-u.quitChan:
			return
		}
	}
}

// handleMessage 处理接收到的消息
func (u *User) handleMessage(messageData []byte) {
	// 解析消息
	msg, err := message.FromJSON(messageData)
	if err != nil {
		log.Printf("解析消息失败: %v\n", err)
		return
	}

	// 处理心跳消息
	if msg.Type == message.HeartbeatMessage {
		u.handleHeartbeat()
		return
	}

	// 设置消息的发送者和房间ID
	msg.SetSender(u.Name)
	msg.SetRoomID(u.roomID)

	// 获取用户当前所在的房间
	if u.roomID != "" {
		// 通过RoomManager广播消息
		if room, exists := roomManager.GetRoom(u.roomID); exists {
			room.Broadcast(msg)
		}
	}
}

// handleHeartbeat 处理心跳消息
func (u *User) handleHeartbeat() {
	u.heartbeatMu.Lock()
	defer u.heartbeatMu.Unlock()

	u.lastHeartbeat = time.Now()
	u.heartbeatFailed = 0
}

// checkHeartbeat 检查心跳状态
func (u *User) checkHeartbeat() {
	u.heartbeatMu.Lock()
	defer u.heartbeatMu.Unlock()

	if time.Since(u.lastHeartbeat) > heartbeatInterval {
		u.heartbeatFailed++
		if u.heartbeatFailed >= heartbeatTimeout {
			// 心跳超时，从房间中移除用户
			if u.roomID != "" {
				if room, exists := roomManager.GetRoom(u.roomID); exists {
					room.RemoveUser(u.ID)
				}
			}
			// 关闭连接
			u.Stop()
		}
	}
}

// NewUserManager 创建用户管理器
func NewUserManager() common.UserManager {
	return &UserManager{}
}

// SetRoomManager 设置房间管理器
func SetRoomManager(rm *room.RoomManager) {
	roomManager = rm
}

// AddUser 添加用户
func (um *UserManager) AddUser(user common.User) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	_, exists := um.users.Load(user.GetID())
	if exists {
		return errors.New("user already exists")
	}

	um.users.Store(user.GetID(), user)
	return nil
}

// RemoveUser 移除用户
func (um *UserManager) RemoveUser(userID string) {
	um.users.Delete(userID)
}

// GetUser 获取用户
func (um *UserManager) GetUser(userID string) (common.User, error) {
	user, exists := um.users.Load(userID)
	if !exists {
		return nil, errors.New("user not found")
	}
	return user.(common.User), nil
}

// BroadcastMessage 广播消息给所有用户
func (um *UserManager) BroadcastMessage(msg []byte) {
	um.users.Range(func(key, value interface{}) bool {
		user := value.(common.User)
		user.SendMessage(msg)
		return true
	})
}
