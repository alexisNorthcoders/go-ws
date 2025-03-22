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

// Room structure to hold room data.
type Room struct {
	id                string
	players           []*websocket.Conn
	snakesMap         map[string]Player
	snakesMapMutex    sync.Mutex
	nextPositionIndex int
	waitingRoom       map[string]Player
	waitingRoomMutex  sync.Mutex
	hasGameStarted    bool
	aliveCount        int
}

type Client struct {
	isConnected bool
	playerId    string
	roomId      string // The room that the player belongs to.
}

func (m FoodUpdateMessage) GetEvent() string {
	return m.Event
}

// Map to store connected clients
var clients = make(map[*websocket.Conn]Client)

var FoodCoordinates [][]int
var clientsMutex sync.Mutex
var serverSnakeCollision = false
var rooms = make(map[string]*Room)
var roomsMutex sync.Mutex

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

	log.Printf("Client connected to room: %s", roomId)

	// Listen for messages from the client
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error or client disconnected:", err)
			room.handleDisconnection(conn)
			break
		}

		// Process the message in the context of the room
		processMessage(conn, msg)
	}
}

func findOrCreateRoom(conn *websocket.Conn, playerId string) string {
	// Try to find an available room with space (max 2 players)
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	for roomId, room := range rooms {
		if len(room.players) < 2 {
			// Add the player to the room
			room.players = append(room.players, conn)
			log.Printf("Player %s joined room %s", playerId, roomId)
			return roomId
		}
	}

	// If no room with space, create a new room
	roomId := generateRoomId()
	rooms[roomId] = &Room{
		id:          roomId,
		players:     []*websocket.Conn{conn},
		waitingRoom: make(map[string]Player),
		snakesMap:   make(map[string]Player),
	}
	log.Printf("Player %s created new room %s", playerId, roomId)
	return roomId
}

func generateRoomId() string {
	return "room_" + fmt.Sprintf("%d", len(rooms)+1)
}

// Process incoming messages
func processMessage(conn *websocket.Conn, msg []byte) {
	var message Message
	err := json.Unmarshal(msg, &message)
	if err != nil {
		log.Println("Error parsing message:", err)
		return
	}

	// Retrieve the room ID of the player
	client := clients[conn]
	roomId := client.roomId

	// Lock the room for safe concurrent access
	roomsMutex.Lock()
	room, exists := rooms[roomId]
	roomsMutex.Unlock()

	if !exists {
		log.Printf("Room %s not found for player %s", roomId, message.Player.ID)
		return
	}

	switch message.Event {
	case "ping":
		clientsMutex.Lock()
		err := conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"pong"}`))
		clientsMutex.Unlock()

		if err != nil {
			log.Println("Error sending pong:", err)
		}

	case "newPlayer":
		room.hasGameStarted = false
		log.Printf("New player joined: %s", message.Player.Name)
		message.Player.Type = "player"
		message.Player.Snake.Speed.X = 1
		message.Player.Snake.Speed.Y = 0
		message.Player.Snake.Tail = []Vector{}
		message.Player.Snake.Size = 0

		room.addToWaitingRoom(message.Player)

	case "waitingRoomStatus":
		log.Println("Sending waiting room status")
		room.broadcastWaitingRoomStatus()

	case "startGame":

		if len(room.waitingRoom) > 0 {
			log.Println("Starting game...")
			room.startGame()
		}
	case "playerMovement":
		log.Printf("Player moved: %s, Id: %s, Key: %s, Position: x: %d, y: %d",
			message.Player.Name, message.Player.ID, message.Key, message.Player.Snake.X, message.Player.Snake.Y)

		if player, exists := room.snakesMap[message.Player.ID]; exists {
			if speed, ok := directionMap[message.Key]; ok {
				// check is snake is trying to move in the same axis
				if (player.Snake.Speed.X != 0 && speed.X != 0) || (player.Snake.Speed.Y != 0 && speed.Y != 0) {
					return
				}
				player.Snake.Speed.X = speed.X
				player.Snake.Speed.Y = speed.Y
				room.snakesMap[message.Player.ID] = player
			}
		}

	case "playerDisconnected":
		log.Printf("Player disconnected: %s", message.ID)
		room.removeFromWaitingRoom(message.ID)
		room.broadcastWaitingRoomStatus()

	case "getConfig":
		log.Println("Client requested game config")
		room.serverSnake()
		sendConfig(conn)

	case "updatePlayer":

		snake, exists := room.waitingRoom[message.Player.ID]
		if !exists {
			log.Println("Player not found in waiting room")
			return
		}
		snake.Colours.Body = message.Player.Colours.Body
		snake.Colours.Head = message.Player.Colours.Head
		snake.Colours.Eyes = message.Player.Colours.Eyes

		room.waitingRoom[message.Player.ID] = snake
		room.broadcastWaitingRoomStatus()

	default:
		log.Println("Unknown event received:", message.Event)
	}
}

func (r *Room) serverSnake() {

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

	r.addToWaitingRoom(serverPlayer)
}

func (r *Room) startGameLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(GameConfigJSON.Fps))
	defer ticker.Stop()

	// ticker for server snake
	moveTicker := time.NewTicker(3 * time.Second)
	defer moveTicker.Stop()

	for {
		select {
		case <-ticker.C: // Main game loop running at FPS rate
			if !r.hasGameStarted {
				println("game not started")
				return
			}

			//	r.snakesMapMutex.Lock()
			r.aliveCount = 0

			for key, player := range r.snakesMap {
				if player.Snake.IsDead {
					println("dead snake")
					continue
				}

				player.Snake.Update(r)
				println("updating player")
				r.snakesMap[key] = player
				r.aliveCount++
			}

			if r.aliveCount <= 1 {
				gameOverMessage := EventMessage{
					Event: "gameover",
				}
				r.broadcast(gameOverMessage)
				r.hasGameStarted = false

				// Reset the snakesMap
				r.snakesMap = make(map[string]Player)

				//	r.snakesMapMutex.Unlock()
				println("gameover event")
				return
			}

			message := SnakeUpdateMessage{
				Event:     "snake_update",
				SnakesMap: r.snakesMap,
			}
			println("snakeupdate broadcast")
			r.broadcast(message)
		//	r.snakesMapMutex.Unlock()

		case <-moveTicker.C: //move server snake every 3 seconds
			r.moveSnake()
		}
	}
}

// Add player to the waiting room
func (r *Room) addToWaitingRoom(player Player) {
	r.waitingRoomMutex.Lock()
	defer r.waitingRoomMutex.Unlock()

	// Assign a starting position
	if r.nextPositionIndex < len(startingPositions) {
		player.Snake.X = startingPositions[r.nextPositionIndex].x*GameConfigJSON.GridSize + GameConfigJSON.LeftSectionSize
		player.Snake.Y = startingPositions[r.nextPositionIndex].y * GameConfigJSON.GridSize
		r.nextPositionIndex++
	} else {
		// Handle case where there are more players than predefined positions
		player.Snake.X = 0 // Default position if needed
		player.Snake.Y = 0
	}

	r.waitingRoom[player.ID] = player
}

// Remove player from the waiting room
func (r *Room) removeFromWaitingRoom(playerID string) {
	r.waitingRoomMutex.Lock()
	delete(r.waitingRoom, playerID)
	r.waitingRoomMutex.Unlock()
}

// Broadcast the current waiting room status
func (r *Room) broadcastWaitingRoomStatus() {
	r.waitingRoomMutex.Lock()
	defer r.waitingRoomMutex.Unlock()
	players := make([]Player, 0, len(r.waitingRoom))
	for _, player := range r.waitingRoom {
		players = append(players, player)
	}

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
func (r *Room) startGame() {
	r.nextPositionIndex = 0
	r.hasGameStarted = true

	message := EventMessage{
		Event: "startGame",
	}

	r.broadcast(message)

	// Move players from waitingRoom to snakesMap
	r.waitingRoomMutex.Lock()
	r.snakesMapMutex.Lock()
	maps.Copy(r.snakesMap, r.waitingRoom)
	r.waitingRoom = make(map[string]Player) // Clear waiting room
	r.waitingRoomMutex.Unlock()
	r.snakesMapMutex.Unlock()

	go r.startGameLoop()
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
func (r *Room) broadcast(message BroadcastMessage) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}

	// Lock the room to prevent concurrent modification of players during broadcast
	r.snakesMapMutex.Lock()
	defer r.snakesMapMutex.Unlock()

	for _, conn := range r.players {
		err := conn.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			log.Println("Error sending message to client:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

func (r *Room) moveSnake() {
	player := r.snakesMap["Server"]
	player.Snake.Speed.X, player.Snake.Speed.Y = getRandomDirection(player.Snake.Speed.X, player.Snake.Speed.Y)
	r.snakesMap["Server"] = player
}

func (r *Room) handleDisconnection(conn *websocket.Conn) {
	log.Printf("Handling disconnection")

	clientsMutex.Lock()
	client, exists := clients[conn]
	if !exists {
		clientsMutex.Unlock()
		log.Println("Client not found in map")
		return
	}
	playerId := client.playerId

	// Remove client from clients map immediately
	delete(clients, conn)
	clientsMutex.Unlock()
	log.Println("Client removed from clients map")

	// Grace period to allow for quick reconnections
	go func() {
		time.Sleep(3 * time.Second) // Grace period

		// Check if the player reconnected
		clientsMutex.Lock()
		for _, c := range clients {
			if c.playerId == playerId {
				// Player reconnected, skip removal
				clientsMutex.Unlock()
				log.Printf("Player %s reconnected, skipping removal", playerId)
				return
			}
		}
		clientsMutex.Unlock()

		// Remove from snakesMap if still disconnected
		r.snakesMapMutex.Lock()
		delete(r.snakesMap, playerId)
		r.snakesMapMutex.Unlock()

		r.removeFromWaitingRoom(playerId)
		r.broadcastWaitingRoomStatus()

		log.Printf("Player %s removed from snakesMap after grace period", playerId)
	}()
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
