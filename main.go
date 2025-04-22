package main

import (
	"chatroom-go/message"
	"chatroom-go/room"
	"chatroom-go/user"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源的WebSocket连接
		},
	}

	userManager = user.NewUserManager()
	roomManager = room.NewRoomManager(userManager)
)

// handleWebSocket 处理WebSocket连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("升级WebSocket连接失败: %v\n", err)
		return
	}

	// 获取用户信息和房间ID
	userID := r.URL.Query().Get("userId")
	userName := r.URL.Query().Get("userName")
	roomID := r.URL.Query().Get("roomId")

	if userID == "" || userName == "" || roomID == "" {
		log.Println("缺少必要的参数")
		conn.Close()
		return
	}

	// 创建新用户
	u := user.NewUser(userID, userName, conn)

	// 获取或创建房间
	chatRoom, exists := roomManager.GetRoom(roomID)
	if !exists {
		chatRoom = roomManager.CreateRoom(roomID, fmt.Sprintf("房间-%s", roomID), userManager)
	}

	// 将用户添加到房间
	err = chatRoom.AddUser(u)
	if err != nil {
		log.Printf("添加用户到房间失败: %v\n", err)
		conn.Close()
		return
	}

	// 启动用户的消息处理
	u.Start()

	// 发送欢迎消息
	welcomeMsg := message.NewMessage(message.SystemMessage, "欢迎来到聊天室！", "系统", roomID)
	if welcomeData, err := welcomeMsg.ToJSON(); err == nil {
		u.SendMessage(welcomeData)
	} else {
		log.Printf("发送欢迎消息失败: %v\n", err)
	}

	// 发送历史消息
	history := chatRoom.GetHistory()
	for _, msg := range history {
		u.SendMessage(msg)
	}
}

// handleCreateRoom 处理创建房间的请求
func handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		RoomID   string `json:"roomId"`
		RoomName string `json:"roomName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	room := roomManager.CreateRoom(data.RoomID, data.RoomName, userManager)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"roomId": room.GetID(),
		"name":   room.GetName(),
	})
}

func main() {
	// 初始化用户管理器和房间管理器
	userManager = user.NewUserManager()
	roomManager = room.NewRoomManager(userManager)
	user.SetRoomManager(roomManager)

	// 设置路由
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/room", handleCreateRoom)

	// 提供静态文件服务
	http.Handle("/", http.FileServer(http.Dir("static")))

	// 创建默认房间
	roomManager.CreateRoom("default", "默认房间", userManager)

	// 启动HTTP服务器
	go func() {
		log.Println("服务器启动在 :8888 端口")
		if err := http.ListenAndServe(":8888", nil); err != nil {
			log.Fatal("服务器启动失败:", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭服务器...")
}
