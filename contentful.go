package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type GameConfig struct {
	EntryTitle       string           `json:"entryTitle"`
	CanvasSize       int              `json:"canvasSize"`
	BackgroundColour ColourDetails    `json:"backgroundColour"`
	FPS              int              `json:"fps"`
	ScaleFactor      int              `json:"scaleFactor"`
	FoodNumber       int              `json:"foodNumber"`
	WaitingRoom      WaitingRoom      `json:"waitingRoom"`
	SnakesCollection SnakesCollection `json:"snakesCollection"`
}

type WaitingRoom struct {
	WaitingMessage   string        `json:"waitingMessage"`
	BackgroundColour ColourDetails `json:"backgroundColour"`
}

type SnakesCollection struct {
	Items []ContentfulSnake `json:"items"`
}

type ContentfulSnake struct {
	Name       string        `json:"name"`
	BodyColour ColourDetails `json:"bodyColour"`
	HeadColour ColourDetails `json:"headColour"`
	EyesColour ColourDetails `json:"eyesColour"`
}

type ColourDetails struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type GameConfigResponse struct {
	Data struct {
		GameConfigCollection struct {
			Items []GameConfig `json:"items"`
		} `json:"gameConfigCollection"`
	} `json:"data"`
}

type SnakeConfigType struct {
	Colours
	Name string `json:"name"`
}

var (
	spaceID     string
	accessToken string
	environment string
)

var GameConfigJSON Config

var SnakeConfig SnakeConfigType

func InitContentful() {

	log.Println("InitContentful() called")

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, using system environment variables")
	}

	spaceID = os.Getenv("CONTENTFUL_SPACE_ID")
	accessToken = os.Getenv("CONTENTFUL_ACCESS_TOKEN")
	environment = os.Getenv("CONTENTFUL_ENVIRONMENT")

	if spaceID == "" || accessToken == "" || environment == "" {
		log.Fatal("Missing required environment variables for Contentful")
	}

	LoadConfig()
}

func LoadQuery(filePath string) (string, error) {
	query, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read query file: %w", err)
	}
	return string(query), nil
}

func FetchGameConfig() ([]GameConfig, error) {

	query, err := LoadQuery("queries/gameConfig.graphql")
	if err != nil {
		log.Printf("Error loading query file: %v", err)
		return nil, err
	}

	body := map[string]interface{}{
		"query": query,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("Error marshaling query to JSON: %v", err)
		return nil, err
	}

	url := fmt.Sprintf("https://graphql.contentful.com/content/v1/spaces/%s/environments/%s", spaceID, environment)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error creating new HTTP request: %v", err)
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request to Contentful: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var respData GameConfigResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		log.Printf("Error decoding response: %v", err)
		return nil, err
	}
	log.Println(respData)

	if len(respData.Data.GameConfigCollection.Items) == 0 {
		log.Println("No items found in the response.")
		return nil, fmt.Errorf("no game configurations found")
	}
	return respData.Data.GameConfigCollection.Items, nil
}

func LoadConfig() {

	configs, err := FetchGameConfig()
	if err != nil {
		log.Println("Failed to fetch ContentfulConfig:", err)

		GameConfigJSON = Config{FoodStorage: 11, Side: 800, Fps: 10}
		return
	}

	contentfulConfig := configs[0]

	GameConfigJSON = Config{
		Side:             contentfulConfig.CanvasSize,
		FoodStorage:      contentfulConfig.FoodNumber,
		Fps:              contentfulConfig.FPS,
		BackgroundColour: contentfulConfig.BackgroundColour.Value,
		ScaleFactor:      contentfulConfig.ScaleFactor,
		WaitingRoom: struct {
			WaitingMessage   string `json:"waitingRoomMessage"`
			BackgroundColour string `json:"backgroundColour"`
		}{
			WaitingMessage:   contentfulConfig.WaitingRoom.WaitingMessage,
			BackgroundColour: contentfulConfig.WaitingRoom.BackgroundColour.Value,
		},
	}

	SnakeConfig = SnakeConfigType{
		Colours: Colours{
			Head: contentfulConfig.SnakesCollection.Items[0].HeadColour.Value,
			Body: contentfulConfig.SnakesCollection.Items[0].BodyColour.Value,
			Eyes: contentfulConfig.SnakesCollection.Items[0].EyesColour.Value,
		},
		Name: contentfulConfig.SnakesCollection.Items[0].Name,
	}
	fmt.Printf("Loaded Contentful Config: %+v\n", GameConfigJSON)
}
