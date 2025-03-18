package main

import (
	"fmt"
	"math/rand"
)

type Vector struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Snake struct {
	X      int      `json:"x"`
	Y      int      `json:"y"`
	Speed  Vector   `json:"speed"`
	Tail   []Vector `json:"tail"`
	Size   int      `json:"size"`
	IsDead bool     `json:"isDead"`
	Score  int      `json:"score"`
	Type   string   `json:"type"`
}

// Update moves the snake and shifts its tail
func (s *Snake) Update() {
	if s.IsDead {
		return
	}
	// Check if snake's head collides with any food
	for i := range FoodCoordinates {
		if s.X == FoodCoordinates[i][0] && s.Y == FoodCoordinates[i][1] {
			s.Size++
			s.Tail = append(s.Tail, Vector{X: s.X, Y: s.Y})
			fmt.Printf("Food eaten x:%d y:%d\n", s.X, s.Y)

			s.Score += 50

			newCoord := [][]int{{rand.Intn(GameConfigJSON.ScaleFactor)*GameConfigJSON.GridSize + GameConfigJSON.LeftSectionSize, rand.Intn(GameConfigJSON.ScaleFactor) * GameConfigJSON.GridSize, FoodCoordinates[i][2]}}
			FoodCoordinates[i] = newCoord[0]

			foodMessage := FoodUpdateMessage{
				Event: "updateFood",
				Food:  newCoord,
			}
			broadcast(foodMessage)
			break
		}
	}

	if s.Size == len(s.Tail) {
		for i := range len(s.Tail) - 1 {
			s.Tail[i] = s.Tail[i+1]
		}
	}

	// Add current position to the end of the tail
	if s.Size > 0 {
		s.Tail[s.Size-1] = Vector{X: s.X, Y: s.Y}
	}

	// Move the snake
	s.X += s.Speed.X * GameConfigJSON.GridSize
	s.Y += s.Speed.Y * GameConfigJSON.GridSize

	side := GameConfigJSON.Side
	leftSectionSize := GameConfigJSON.LeftSectionSize

	if s.X >= side+leftSectionSize {
		s.X = 0 + leftSectionSize
	} else if s.X < 0+leftSectionSize {
		s.X = side + leftSectionSize
	}

	if s.Y >= side {
		s.Y = 0
	} else if s.Y < 0 {
		s.Y = side
	}

	// Check for self-collision
	for _, segment := range s.Tail {
		if s.X == segment.X && s.Y == segment.Y {
			fmt.Println("Game over! Snake collided with itself.")
			s.IsDead = true
			break
		}
	}

	// Check for collision with other snakes' tails
	for _, otherSnake := range snakesMap {
		if (otherSnake.Snake.X == s.X && otherSnake.Snake.Y == s.Y) || otherSnake.Snake.IsDead || (otherSnake.Snake.Type == "server" && !serverSnakeCollision) {
			continue
		}

		for _, segment := range otherSnake.Snake.Tail {
			if s.Type == "server" && !serverSnakeCollision {
				return
			}
			if s.X == segment.X && s.Y == segment.Y {
				fmt.Printf("Game over! Snake %p collided with another snake's tail.\n", s)
				s.IsDead = true
				return
			}
		}
	}
}
