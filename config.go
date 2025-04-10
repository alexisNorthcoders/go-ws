package main

import (
	"math/rand"
)

type Config struct {
	BackgroundNumber int    `json:"backgroundNumber"`
	Side             int    `json:"side"`
	LeftSectionSize  int    `json:"leftSectionSize"`
	FoodStorage      int    `json:"foodStorage"`
	Fps              int    `json:"fps"`
	BackgroundColour string `json:"backgroundColour"`
	ScaleFactor      int    `json:"scaleFactor"`
	GridSize         int    `json:"gridSize"`
	WaitingRoom      struct {
		WaitingMessage   string `json:"waitingRoomMessage"`
		BackgroundColour string `json:"backgroundColour"`
	} `json:"waitingRoom"`
}

func GenerateFoodCoordinates(foodCount int) [][]any {
	coordinates := make([][]any, foodCount)

	for i := range foodCount {
		x := rand.Intn(GameConfigJSON.ScaleFactor)
		y := rand.Intn(GameConfigJSON.ScaleFactor)

		typeIndex := rand.Intn(len(foodTypes))
		coordinates[i] = []any{x, y, i, foodTypes[typeIndex]}
	}

	return coordinates
}

var directionMap = map[string]struct{ X, Y int }{
	"l": {X: -1, Y: 0},
	"r": {X: 1, Y: 0},
	"u": {X: 0, Y: -1},
	"d": {X: 0, Y: 1},
}
