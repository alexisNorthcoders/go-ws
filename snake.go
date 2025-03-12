package main

import (
	"fmt"
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
}

// Update moves the snake and shifts its tail
func (s *Snake) Update() {
	if s.IsDead {
		return
	}

	if s.Size == len(s.Tail) {
		for i := 0; i < len(s.Tail)-1; i++ {
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

	// Check if snake's head collides with any food
	for i := 0; i < len(FoodCoordinates); i++ {

		if s.X == FoodCoordinates[i][0] && s.Y == FoodCoordinates[i][1] {
			s.Size++
			s.Tail = append(s.Tail, Vector{X: s.X, Y: s.Y})
			fmt.Printf("Food eaten x:%d y:%d", s.X, s.Y)
			break
		}
	}
}
