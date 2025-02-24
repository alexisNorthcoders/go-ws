package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
)

type Config struct {
	Side        int `json:"side"`
	FoodStorage int `json:"foodStorage"`
	Fps         int `json:"fps"`
}

var GameConfig Config

func LoadConfig(filename string) {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Println("Using default config, error reading file:", err)
		GameConfig = Config{FoodStorage: 50, Side: 800, Fps: 10}
		return
	}

	err = json.Unmarshal(file, &GameConfig)
	if err != nil {
		log.Println("Error parsing config file:", err)
		GameConfig = Config{FoodStorage: 50, Side: 800, Fps: 10}
	}
}

func GenerateFoodCoordinates(foodCount int) [][]int {
	coordinates := make([][]int, foodCount)

	for i := range foodCount {
		coordinates[i] = []int{rand.Intn(20), rand.Intn(20)}
	}

	return coordinates
}

func init() {
	LoadConfig("config.json")
	fmt.Printf("Loaded Config: %+v\n", GameConfig)
}
