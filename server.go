package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

// Player structure
type Player struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Message structure
type Message struct {
	Event  string `json:"event"`
	Player Player `json:"player,omitempty"`
	Key    string `json:"key,omitempty"`
	ID     string `json:"id,omitempty"`
}

// Map to store connected clients
var clients = make(map[*websocket.Conn]bool)
var clientsMutex sync.Mutex

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	// Add client to the map
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	log.Println("Client connected")

	for {
		// Read message from client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error or client disconnected:", err)
			handleDisconnection(conn)
			break
		}

		// Process message
		processMessage(conn, msg)
	}
}

// Process incoming messages
func processMessage(conn *websocket.Conn, msg []byte) {
	var message Message
	err := json.Unmarshal(msg, &message)
	if err != nil {
		log.Println("Error parsing message:", err)
		return
	}

	switch message.Event {
	case "newPlayer":
		log.Printf("New player joined: %s", message.Player.Name)
		broadcast(message)

	case "playerMovement":
		log.Printf("Player moved: %s, Key: %s", message.Player.Name, message.Key)
		broadcast(message)

	case "playerDisconnected":
		log.Printf("Player disconnected: %s", message.ID)
		broadcast(message)

	default:
		log.Println("Unknown event received:", message.Event)
	}
}

// Broadcast message to all connected clients
func broadcast(message Message) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			log.Println("Error sending message to client:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

// Handle client disconnection
func handleDisconnection(conn *websocket.Conn) {
	clientsMutex.Lock()
	delete(clients, conn)
	clientsMutex.Unlock()

	// Notify others that a player has disconnected
	message := Message{
		Event: "playerDisconnected",
		ID:    "some-unique-id", // You need to track player IDs properly
	}
	broadcast(message)
}

func main() {
	http.HandleFunc("/ws", handleConnections)

	port := "4001"
	log.Println("WebSocket server started on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
