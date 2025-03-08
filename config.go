package main

import (
	"math/rand"
)

type Config struct {
	Side        int `json:"side"`
	FoodStorage int `json:"foodStorage"`
	Fps         int `json:"fps"`
}

func GenerateFoodCoordinates(foodCount int) [][]int {
	coordinates := make([][]int, foodCount)

	for i := range foodCount {
		coordinates[i] = []int{rand.Intn(20), rand.Intn(20), i}
	}

	return coordinates
}
