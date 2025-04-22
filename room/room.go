package room

import (
	"chatroom-go/common"
	"chatroom-go/message"
	"fmt"
	"sync"
)

// Room 实现common.Room接口
type Room struct {
	ID      string             // 房间唯一标识
	Name    string             // 房间名称
	Users   common.UserManager // 房间内的用户管理器
	History [][]byte           // 消息历史记录
	mu      sync.RWMutex       // 读写锁
}

// RoomManager 房间管理器 实现common.RoomManager接口
type RoomManager struct {
	rooms sync.Map // 存储所有房间
}

// NewRoom 创建新房间
func NewRoom(id, name string, userManager common.UserManager) *Room {
	return &Room{
		ID:      id,
		Name:    name,
		Users:   userManager,
		History: make([][]byte, 0),
	}
}

// GetID 获取房间ID
func (r *Room) GetID() string {
	return r.ID
}

// GetName 获取房间名称
func (r *Room) GetName() string {
	return r.Name
}

// AddUser 添加用户到房间
func (r *Room) AddUser(u common.User) error {
	// 先更新用户房间ID
	u.SetRoomID(r.ID)

	// 准备加入消息
	joinMsg := message.NewMessage(message.UserJoinMessage, u.GetName()+" 加入了房间", "系统", r.ID)
	joinData, _ := joinMsg.ToJSON()

	r.mu.Lock()
	defer r.mu.Unlock()

	// 将用户添加到房间
	if err := r.Users.AddUser(u); err != nil {
		return err
	}

	// 广播消息
	r.Users.BroadcastMessage(joinData)
	fmt.Println("AddUser", joinMsg)

	return nil
}

// RemoveUser 从房间移除用户
func (r *Room) RemoveUser(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 获取用户信息
	u, err := r.Users.GetUser(userID)
	if err == nil {
		// 发送用户离开消息
		leaveMsg := message.NewMessage(message.UserLeaveMessage, u.GetName()+" 离开了房间", "系统", r.ID)
		leaveData, _ := leaveMsg.ToJSON()
		r.Broadcast(leaveData)

		// 移除用户
		r.Users.RemoveUser(userID)
	}
}

// Broadcast 广播消息到房间内所有用户
func (r *Room) Broadcast(msg []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 保存消息到历史记录
	r.History = append(r.History, msg)

	// 广播消息给所有用户
	r.Users.BroadcastMessage(msg)
}

// GetHistory 获取房间的消息历史记录
func (r *Room) GetHistory() [][]byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.History
}

// NewRoomManager 创建房间管理器
func NewRoomManager(userManager common.UserManager) *RoomManager {
	return &RoomManager{}
}

// CreateRoom 创建新房间
func (rm *RoomManager) CreateRoom(id, name string, userManager common.UserManager) common.Room {
	room := NewRoom(id, name, userManager)
	rm.rooms.Store(id, room)
	return room
}

// GetRoom 获取房间
func (rm *RoomManager) GetRoom(roomID string) (common.Room, bool) {
	room, exists := rm.rooms.Load(roomID)
	if !exists {
		return nil, false
	}
	return room.(*Room), true
}

// DeleteRoom 删除房间
func (rm *RoomManager) DeleteRoom(roomID string) {
	rm.rooms.Delete(roomID)
}
