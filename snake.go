package main

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
	Scale  int      `json:"scale"`
}

// Update moves the snake and shifts its tail
func (s *Snake) Update() {
	if s.IsDead {
		return
	}

	// If no food was eaten, shift the tail elements forward
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
	s.X += s.Speed.X * s.Scale
	s.Y += s.Speed.Y * s.Scale
}
