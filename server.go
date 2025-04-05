package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

type BroadcastMessage interface {
	GetEvent() string
}

type Colours struct {
	Body string `json:"body"`
	Head string `json:"head"`
	Eyes string `json:"eyes"`
}

type Player struct {
	Name    string  `json:"name"`
	ID      string  `json:"id"`
	Snake   Snake   `json:"snake,omitempty"`
	Colours Colours `json:"colours,omitempty"`
	Type    string  `json:"type,omitempty"`
}

type Client struct {
	isConnected bool
	playerId    string
	roomId      string // The room that the player belongs to.
}

// Map to store connected clients
var clients = make(map[*websocket.Conn]Client)

var clientsMutex sync.Mutex
var serverSnakeCollision = false

func handleConnections(w http.ResponseWriter, req *http.Request) {
	// Extract the playerId from the URL query parameters
	playerId := req.URL.Query().Get("playerId")
	if playerId == "" {
		log.Println("Player ID is missing in the URL")
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("Player ID extracted: %s", playerId)

	// Upgrade HTTP request to WebSocket
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	// Check if player already has a connection
	clientsMutex.Lock()
	for existingConn, client := range clients {
		if client.playerId == playerId {
			log.Printf("Duplicate connection detected for player: %s", playerId)
			existingConn.Close()
			delete(clients, existingConn)
		}
	}
	clientsMutex.Unlock()

	// Find or create a room for the player
	roomId := findOrCreateRoom(conn, playerId)

	// Lock the room and add the client
	roomsMutex.Lock()
	room, exists := rooms[roomId]
	roomsMutex.Unlock()

	if !exists {
		log.Println("Failed to find or create room")
		return
	}

	clientsMutex.Lock()
	clients[conn] = Client{
		isConnected: true,
		playerId:    playerId,
		roomId:      roomId,
	}
	clientsMutex.Unlock()

	log.Printf("Client %s connected to room: %s", clients[conn].playerId, roomId)

	// Create a channel for received messages
	messageChannel := make(chan []byte)

	// Goroutine to read messages from the WebSocket
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error or client disconnected:", err)
				room.handleDisconnection(conn)
				break
			}
			messageChannel <- msg
		}
	}()

	go func() {
		for msg := range messageChannel {
			processMessage(conn, msg)
		}
	}()

	select {}
}

// Process incoming messages
func processMessage(conn *websocket.Conn, msg []byte) {

	// Check if the message is a simple string before attempting to unmarshal it
	strMsg := string(msg)
	// p for ping
	if strMsg == "p" {
		clientsMutex.Lock()
		err := conn.WriteMessage(websocket.TextMessage, []byte("p"))
		clientsMutex.Unlock()

		if err != nil {
			log.Println("Error sending pong:", err)
		}
		return
	}

	var message Message

	// Retrieve the room ID of the player
	clientsMutex.Lock()
	client := clients[conn]
	roomId := client.roomId
	clientsMutex.Unlock()
	roomsMutex.Lock()
	room, exists := rooms[roomId]
	roomsMutex.Unlock()

	if !exists {
		log.Printf("Room %s not found for player %s", roomId, message.Player.ID)
		return
	}

	if strings.HasPrefix(strMsg, "m:") {
		parts := strings.Split(strMsg[2:], ":")
		playerId := parts[0]
		key := parts[1]

		roomsMutex.Lock()
		if player, exists := room.snakesMap[playerId]; exists {
			if speed, ok := directionMap[key]; ok {
				// check is snake is trying to move in the same axis
				if (player.Snake.Speed.X != 0 && speed.X != 0) || (player.Snake.Speed.Y != 0 && speed.Y != 0) {
					roomsMutex.Unlock()
					return
				}
				player.Snake.Speed.X = speed.X
				player.Snake.Speed.Y = speed.Y
				room.snakesMap[playerId] = player
			}
		}
		roomsMutex.Unlock()
		return
	}

	err := json.Unmarshal(msg, &message)
	if err != nil {
		log.Println("Error parsing message:", err)
		return
	}

	switch message.Event {
	case "newPlayer":
		if !room.hasGameStarted {

			log.Printf("New player joined: %s", message.Player.Name)
			message.Player.Type = "player"
			message.Player.Snake.Speed.X = 1
			message.Player.Snake.Speed.Y = 0
			message.Player.Snake.Tail = []Vector{}
			message.Player.Snake.Size = 0

			roomsMutex.Lock()
			room.addToWaitingRoom(message.Player)
			room.broadcastWaitingRoomStatus()
			room.sendConfig(conn)
			roomsMutex.Unlock()
			log.Printf("Config sent to player %s", client.playerId)
		}
	case "waitingRoomStatus":
		log.Printf("Sending waiting room status to room: %s", roomId)
		room.broadcastWaitingRoomStatus()

	case "startGame":

		if len(room.waitingRoom) > 0 {
			log.Printf("Starting game on room: %s", room.id)
			room.startGame()
		}

	case "updatePlayer":

		snake, exists := room.waitingRoom[message.Player.ID]
		if !exists {
			log.Println("Player not found in waiting room")
			return
		}
		snake.Colours.Body = message.Player.Colours.Body
		snake.Colours.Head = message.Player.Colours.Head
		snake.Colours.Eyes = message.Player.Colours.Eyes

		roomsMutex.Lock()
		room.waitingRoom[message.Player.ID] = snake
		room.broadcastWaitingRoomStatus()
		roomsMutex.Unlock()

	default:
		log.Println("Unknown event received:", message.Event)
	}
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	secret := os.Getenv("WEBHOOK_SECRET")
	if secret != "" && r.Header.Get("X-Gitlab-Token") != secret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Println("Webhook received. Triggering InitContentful()")
	InitContentful()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Webhook received and InitContentful triggered")
}

func main() {
	port := "4002"

	if os.Getenv("BUILD_MODE") == "true" {
		port = "4001"
	}

	InitContentful()

	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/webhook", webhookHandler)

	log.Println("WebSocket server started on port", port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
