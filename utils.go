package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

func getRandomDirection(X int, Y int) (int, int) {

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Filter out the opposite direction and same direction
	validDirections := []struct{ X, Y int }{}
	for _, dir := range directionMap {
		if !(dir.X == -X && dir.Y == -Y) && !(dir.X == X && dir.Y == Y) {
			validDirections = append(validDirections, dir)
		}
	}
	newDirection := validDirections[rng.Intn(len(validDirections))]

	return newDirection.X, newDirection.Y
}

func randomNumber() int {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	return rand.Intn(91) + 1
}

func findOrCreateRoom(conn *websocket.Conn, playerId string) string {
	// Try to find an available room with space (max 2 players)
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	for roomId, room := range rooms {
		if len(room.players) < 2 && !room.hasGameStarted {
			// Add the player to the room
			room.players = append(room.players, conn)
			log.Printf("Player %s joined room %s", playerId, roomId)
			return roomId
		}
	}

	// If no room with space, create a new room
	roomId := generateRoomId()
	rooms[roomId] = &Room{
		id:              roomId,
		players:         []*websocket.Conn{conn},
		waitingRoom:     make(map[string]Player),
		snakesMap:       make(map[string]Player),
		FoodCoordinates: GenerateFoodCoordinates(GameConfigJSON.FoodStorage),
	}
	log.Printf("Player %s created new room: %s", playerId, roomId)
	return roomId
}

func generateRoomId() string {
	return "room_" + fmt.Sprintf("%d", len(rooms)+1)
}
