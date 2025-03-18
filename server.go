package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"maps"

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

type Message struct {
	Event  string  `json:"event"`
	Player Player  `json:"player,omitempty"`
	Key    string  `json:"key,omitempty"`
	ID     string  `json:"id,omitempty"`
	Config *Config `json:"config,omitempty"`
}

type EventMessage struct {
	Event string `json:"event"`
}

func (m EventMessage) GetEvent() string {
	return m.Event
}

type SnakeUpdateMessage struct {
	Event     string            `json:"event"`
	SnakesMap map[string]Player `json:"snakesMap"`
}

func (m SnakeUpdateMessage) GetEvent() string {
	return m.Event
}

type ConfigMessage struct {
	Event  string  `json:"event"`
	Config *Config `json:"config,omitempty"`
	Food   [][]int `json:"food"`
}

func (m ConfigMessage) GetEvent() string {
	return m.Event
}

type FoodUpdateMessage struct {
	Event string  `json:"event"`
	Food  [][]int `json:"food"`
}

func (m FoodUpdateMessage) GetEvent() string {
	return m.Event
}

// Map to store connected clients
var clients = make(map[*websocket.Conn]bool)
var waitingRoom = make(map[string]Player)
var snakesMap = make(map[string]Player) // Store active game players
var snakesMapMutex = &sync.Mutex{}      // Store active snakes
var FoodCoordinates [][]int
var hasGameStarted bool
var clientsMutex sync.Mutex
var waitingRoomMutex sync.Mutex
var serverSnakeCollision = false

var directionMap = map[string]struct{ X, Y int }{
	"LEFT":  {X: -1, Y: 0},
	"RIGHT": {X: 1, Y: 0},
	"UP":    {X: 0, Y: -1},
	"DOWN":  {X: 0, Y: 1},
}

// position vars only 4 positions for now
var startingPositions = []struct{ x, y int }{
	{5, 5}, {15, 5}, {15, 5}, {15, 15},
}
var nextPositionIndex = 0

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
	case "ping":
		conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"pong"}`))

	case "newPlayer":
		hasGameStarted = false
		log.Printf("New player joined: %s", message.Player.Name)
		message.Player.Type = "player"
		message.Player.Snake.Speed.X = 1
		message.Player.Snake.Speed.Y = 0
		message.Player.Snake.Tail = []Vector{}
		message.Player.Snake.Size = 0

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
		log.Printf("Player moved: %s, Id: %s, Key: %s, Position: x: %d, y: %d",
			message.Player.Name, message.Player.ID, message.Key, message.Player.Snake.X, message.Player.Snake.Y)

		if player, exists := snakesMap[message.Player.ID]; exists {
			if speed, ok := directionMap[message.Key]; ok {
				// check is snake is trying to move in the same axis
				if (player.Snake.Speed.X != 0 && speed.X != 0) || (player.Snake.Speed.Y != 0 && speed.Y != 0) {
					return
				}
				player.Snake.Speed.X = speed.X
				player.Snake.Speed.Y = speed.Y
				snakesMap[message.Player.ID] = player
			}
		}

	case "playerDisconnected":
		log.Printf("Player disconnected: %s", message.ID)
		removeFromWaitingRoom(message.ID)
		broadcastWaitingRoomStatus()

	case "getConfig":
		log.Println("Client requested game config")
		serverSnake()
		sendConfig(conn)

	case "updatePlayer":

		snake, exists := waitingRoom[message.Player.ID]
		if !exists {
			log.Println("Player not found in waiting room")
			return
		}
		snake.Colours.Body = message.Player.Colours.Body
		snake.Colours.Head = message.Player.Colours.Head
		snake.Colours.Eyes = message.Player.Colours.Eyes

		waitingRoom[message.Player.ID] = snake
		broadcastWaitingRoomStatus()

	default:
		log.Println("Unknown event received:", message.Event)
	}
}

func serverSnake() {

	snake := Snake{
		Speed: Vector{X: 1, Y: 0},
		Tail:  []Vector{},
		Type:  "server",
	}

	serverPlayer := Player{
		Name:    SnakeConfig.Name,
		ID:      "Server",
		Colours: SnakeConfig.Colours,
		Type:    "server",
		Snake:   snake,
	}

	addToWaitingRoom(serverPlayer)
}

func startGameLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(GameConfigJSON.Fps))
	defer ticker.Stop()

	// ticker for server snake
	moveTicker := time.NewTicker(3 * time.Second)
	defer moveTicker.Stop()

	for {
		select {
		case <-ticker.C: // Main game loop running at FPS rate
			if !hasGameStarted {
				return
			}

			snakesMapMutex.Lock()
			aliveCount := 0

			for key, player := range snakesMap {
				if player.Snake.IsDead {
					continue
				}

				player.Snake.Update()
				snakesMap[key] = player
				aliveCount++
			}

			if aliveCount <= 1 {
				gameOverMessage := EventMessage{
					Event: "gameover",
				}
				broadcast(gameOverMessage)
				hasGameStarted = false
				snakesMapMutex.Unlock()
				return
			}

			message := SnakeUpdateMessage{
				Event:     "snake_update",
				SnakesMap: snakesMap,
			}
			broadcast(message)
			snakesMapMutex.Unlock()

		case <-moveTicker.C: //move server snake every 3 seconds
			moveSnake()
		}
	}
}

// Add player to the waiting room
func addToWaitingRoom(player Player) {
	waitingRoomMutex.Lock()
	defer waitingRoomMutex.Unlock()

	// Assign a starting position
	if nextPositionIndex < len(startingPositions) {
		player.Snake.X = startingPositions[nextPositionIndex].x*GameConfigJSON.GridSize + GameConfigJSON.LeftSectionSize
		player.Snake.Y = startingPositions[nextPositionIndex].y * GameConfigJSON.GridSize
		nextPositionIndex++
	} else {
		// Handle case where there are more players than predefined positions
		player.Snake.X = 0 // Default position if needed
		player.Snake.Y = 0
	}

	waitingRoom[player.ID] = player
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
	nextPositionIndex = 0
	hasGameStarted = true

	message := EventMessage{
		Event: "startGame",
	}

	broadcast(message)

	// Move players from waitingRoom to snakesMap
	waitingRoomMutex.Lock()
	snakesMapMutex.Lock()
	maps.Copy(snakesMap, waitingRoom)
	waitingRoom = make(map[string]Player) // Clear waiting room
	waitingRoomMutex.Unlock()
	snakesMapMutex.Unlock()

	go startGameLoop()
}

func sendConfig(conn *websocket.Conn) {
	fmt.Println(FoodCoordinates)

	configMessage := ConfigMessage{
		Event:  "config",
		Config: &GameConfigJSON,
		Food:   FoodCoordinates,
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
func broadcast(message BroadcastMessage) {
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

func moveSnake() {
	player := snakesMap["Server"]
	player.Snake.Speed.X, player.Snake.Speed.Y = getRandomDirection(player.Snake.Speed.X, player.Snake.Speed.Y)
	snakesMap["Server"] = player
}

func handleDisconnection(conn *websocket.Conn) {
	clientsMutex.Lock()
	delete(clients, conn)
	clientsMutex.Unlock()
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
