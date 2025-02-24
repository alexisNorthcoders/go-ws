package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func init() {
	LoadConfig("config.json")
	fmt.Printf("Loaded Config: %+v\n", GameConfig)
}
