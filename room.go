package main

import (
	"encoding/json"
	"log"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Room structure to hold room data.
type Room struct {
	id                string
	playersMutex      sync.Mutex
	players           []*websocket.Conn
	snakesMap         map[string]Player
	snakesMapMutex    sync.Mutex
	nextPositionIndex int
	waitingRoom       map[string]Player
	waitingRoomMutex  sync.Mutex
	hasGameStarted    bool
	aliveCount        int
	FoodCoordinates   [][]int
}

var rooms = make(map[string]*Room)
var roomsMutex sync.Mutex

// position vars only 4 positions for now
var startingPositions = []struct{ x, y int }{
	{5, 5}, {15, 5}, {15, 5}, {15, 15},
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

			/* r.snakesMapMutex.Lock()
			defer r.snakesMapMutex.Unlock() */

			r.aliveCount = 0

			for key, player := range r.snakesMap {
				if player.Snake.IsDead {
					continue
				}

				player.Snake.Update(r)
				r.snakesMap[key] = player
				r.aliveCount++
			}

			if r.aliveCount <= 0 {
				gameOverMessage := EventMessage{
					Event: "gameover",
				}
				r.broadcast(gameOverMessage)
				r.hasGameStarted = false
				r.players = nil
				r.snakesMap = nil
				r.FoodCoordinates = nil

				roomID := r.id
				log.Printf("Deleting room %s\n", roomID)
				delete(rooms, roomID)

				return
			}

			message := SnakeUpdateMessage{
				Event:     "snake_update",
				SnakesMap: r.snakesMap,
			}
			r.broadcast(message)

		case <-moveTicker.C: //move server snake every 3 seconds
			//	r.moveSnake()
		}
	}
}

// Add player to the waiting room
func (r *Room) addToWaitingRoom(player Player) {
	r.waitingRoomMutex.Lock()
	defer r.waitingRoomMutex.Unlock()

	// Assign a starting position
	if r.nextPositionIndex < len(startingPositions) {
		player.Snake.X = startingPositions[r.nextPositionIndex].x
		player.Snake.Y = startingPositions[r.nextPositionIndex].y
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

	for _, conn := range r.players {
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

func (r *Room) sendConfig(conn *websocket.Conn) {

	GameConfigJSON.BackgroundNumber = randomNumber()
	configMessage := ConfigMessage{
		Event:  "config",
		Config: &GameConfigJSON,
		Food:   r.FoodCoordinates,
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

	playersCopy := slices.Clone(r.players)

	for i, conn := range playersCopy {

		err := conn.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			log.Println("Error sending message to client:", err)

			r.playersMutex.Lock()
			conn.Close()
			delete(clients, conn)
			r.players = slices.Delete(r.players, i, i+1)
			r.playersMutex.Unlock()

			log.Printf("Player at index %d removed from the game due to an error.\n", i)
		}
	}
}

func (r *Room) moveSnake() {
	player := r.snakesMap["Server"]
	player.Snake.Speed.X, player.Snake.Speed.Y = getRandomDirection(player.Snake.Speed.X, player.Snake.Speed.Y)
	r.snakesMap["Server"] = player
}

func (r *Room) handleDisconnection(conn *websocket.Conn) {

	r.removePlayerConnection(conn)

	clientsMutex.Lock()
	client, exists := clients[conn]
	if !exists {
		clientsMutex.Unlock()
		log.Println("Client not found in map")
		return
	}
	playerId := client.playerId

	delete(clients, conn)
	clientsMutex.Unlock()
	log.Println("Client removed from clients map")

	r.snakesMapMutex.Lock()
	delete(r.snakesMap, playerId)
	r.snakesMapMutex.Unlock()

	if !r.hasGameStarted {
		r.removeFromWaitingRoom(playerId)
		r.broadcastWaitingRoomStatus()
	}
}

func (r *Room) removePlayerConnection(conn *websocket.Conn) {

	r.playersMutex.Lock()
	defer r.playersMutex.Unlock()

	for i, playerConn := range r.players {
		if playerConn == conn {

			r.players = slices.Delete(r.players, i, i+1)
			log.Printf("Removed connection from players in room %s", r.id)
			return
		}
	}
	log.Printf("Connection not found in players list for room %s", r.id)
}
