package main

type Message struct {
	Event  string  `json:"event"`
	Player Player  `json:"player,omitempty"`
	Key    string  `json:"key,omitempty"`
	ID     string  `json:"id,omitempty"`
	Config *Config `json:"config,omitempty"`
}

type EventMessage struct {
	Event string `json:"event"`
}

func (m EventMessage) GetEvent() string {
	return m.Event
}

type SnakeUpdateMessage struct {
	Event     string            `json:"event"`
	SnakesMap map[string]Player `json:"snakesMap"`
}

func (m SnakeUpdateMessage) GetEvent() string {
	return m.Event
}

type ConfigMessage struct {
	Event  string  `json:"event"`
	Config *Config `json:"config,omitempty"`
	Food   [][]int `json:"food"`
}

func (m ConfigMessage) GetEvent() string {
	return m.Event
}

type FoodUpdateMessage struct {
	Event string  `json:"event"`
	Food  [][]int `json:"food"`
}

func (m FoodUpdateMessage) GetEvent() string {
	return m.Event
}
