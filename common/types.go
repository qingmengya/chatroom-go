package common

// User 定义用户接口
type User interface {
	GetID() string
	GetName() string
	GetRoomID() string
	SetRoomID(roomID string)
	SendMessage(msg []byte)
}

// UserManager 定义用户管理器接口
type UserManager interface {
	AddUser(user User) error
	RemoveUser(userID string)
	GetUser(userID string) (User, error)
	BroadcastMessage(msg []byte)
}

// Room 定义房间接口
type Room interface {
	GetID() string
	GetName() string
	AddUser(user User) error
	RemoveUser(userID string)
	Broadcast(msg []byte)
	GetHistory() [][]byte
}

// RoomManager 定义房间管理器接口
type RoomManager interface {
	CreateRoom(id, name string, user UserManager) Room
	GetRoom(roomID string) (Room, bool)
	DeleteRoom(roomID string)
}
