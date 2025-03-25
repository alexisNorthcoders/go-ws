package main

import (
	"math/rand"
	"time"
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
