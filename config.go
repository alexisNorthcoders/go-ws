package main

import (
	"math/rand"
)

type Config struct {
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

func GenerateFoodCoordinates(foodCount int) [][]int {
	coordinates := make([][]int, foodCount)

	for i := range foodCount {
		coordinates[i] = []int{rand.Intn(GameConfigJSON.ScaleFactor)*GameConfigJSON.GridSize + GameConfigJSON.LeftSectionSize, rand.Intn(GameConfigJSON.ScaleFactor) * GameConfigJSON.GridSize, i}
	}

	return coordinates
}
