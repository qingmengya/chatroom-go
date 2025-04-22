package message

import (
	"encoding/json"
	"time"
)

// MessageType 定义消息类型
type MessageType int

const (
	TextMessage      MessageType = iota // 文本消息
	SystemMessage                       // 系统消息
	UserJoinMessage                     // 用户加入消息
	UserLeaveMessage                    // 用户离开消息
)

// Message 定义消息的基本结构
type Message struct {
	Type      MessageType `json:"type"`      // 消息类型
	Content   string      `json:"content"`   // 消息内容
	From      string      `json:"from"`      // 发送者
	Timestamp time.Time   `json:"timestamp"` // 发送时间
	RoomID    string      `json:"roomId"`    // 房间ID
}

// MessageHandler 定义消息处理器接口
type MessageHandler interface {
	Handle(msg *Message) error
}

// NewMessage 创建新消息
func NewMessage(msgType MessageType, content, from, roomID string) *Message {
	return &Message{
		Type:      msgType,
		Content:   content,
		From:      from,
		Timestamp: time.Now(),
		RoomID:    roomID,
	}
}

// ToJSON 将消息转换为JSON字符串
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON 从JSON字符串解析消息
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// SetSender 设置消息发送者
func (m *Message) SetSender(sender string) {
	m.From = sender
}

// SetRoomID 设置消息所属房间ID
func (m *Message) SetRoomID(roomID string) {
	m.RoomID = roomID
}
