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
	Event  string  `json:"event"`
	Player Player  `json:"player,omitempty"`
	Key    string  `json:"key,omitempty"`
	ID     string  `json:"id,omitempty"`
	Config *Config `json:"config,omitempty"`
	Food   [][]int `json:"food,omitempty"`
}

// Map to store connected clients
var clients = make(map[*websocket.Conn]bool)
var waitingRoom = make(map[string]Player)
var clientsMutex sync.Mutex
var waitingRoomMutex sync.Mutex

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
		addToWaitingRoom(message.Player)
	case "waitingRoomStatus":
		log.Println("Sending waiting room status")
		broadcastWaitingRoomStatus()

	case "startGame":
		if len(waitingRoom) > 0 {
			log.Println("Starting game...")
			startGame()
		}

	case "playerMovement":
		log.Printf("Player moved: %s, Key: %s", message.Player.Name, message.Key)
		broadcast(message)

	case "playerDisconnected":
		log.Printf("Player disconnected: %s", message.ID)
		removeFromWaitingRoom(message.ID)
		broadcastWaitingRoomStatus()
		broadcast(message)

	case "getConfig":
		log.Println("Client requested game config")
		sendConfig(conn)

	case "spawnFood":
		coords := GenerateFoodCoordinates(1)
		log.Printf("Spawning food at: %v", coords)

		foodMessage := Message{
			Event: "spawnFood",
			Food:  coords,
		}

		broadcast(foodMessage)

	default:
		log.Println("Unknown event received:", message.Event)
	}
}

// Add player to the waiting room
func addToWaitingRoom(player Player) {
	waitingRoomMutex.Lock()
	waitingRoom[player.Name] = player
	waitingRoomMutex.Unlock()
}

// Remove player from the waiting room
func removeFromWaitingRoom(playerID string) {
	waitingRoomMutex.Lock()
	delete(waitingRoom, playerID)
	waitingRoomMutex.Unlock()
}

// Broadcast the current waiting room status
func broadcastWaitingRoomStatus() {
	waitingRoomMutex.Lock()
	players := make([]Player, 0, len(waitingRoom))
	for _, player := range waitingRoom {
		players = append(players, player)
	}
	waitingRoomMutex.Unlock()

	// Assign players to JSON and include in the message
	messageBytes, err := json.Marshal(struct {
		Event   string   `json:"event"`
		Players []Player `json:"players"`
	}{
		Event:   "waitingRoomStatus",
		Players: players,
	})
	if err != nil {
		log.Println("Error encoding waiting room status:", err)
		return
	}

	// Broadcast the updated waiting room status
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, messageBytes)
		if err != nil {
			log.Println("Error sending waiting room status to client:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

// Start the game when all players are ready
func startGame() {

	message := Message{
		Event: "startGame",
	}

	broadcast(message)

	// Clear the waiting room since the game has started
	waitingRoomMutex.Lock()
	waitingRoom = make(map[string]Player)
	waitingRoomMutex.Unlock()
}

func sendConfig(conn *websocket.Conn) {
	coords := GenerateFoodCoordinates(GameConfig.FoodStorage)

	configMessage := Message{
		Event:  "config",
		Config: &GameConfig,
		Food:   coords,
	}
	msgBytes, err := json.Marshal(configMessage)
	if err != nil {
		log.Println("Error encoding config message:", err)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		log.Println("Error sending config to client:", err)
		conn.Close()
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
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
