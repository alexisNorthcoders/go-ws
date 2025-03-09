# Snake Game WebSocket Server

This is a WebSocket server written in Go that facilitates real-time communication between players in a multiplayer Snake game (https://github.com/alexisNorthcoders/p5_snake_game).

## Features
- Handles WebSocket connections using Gorilla WebSocket.
- Supports player connections and disconnections.
- Processes and broadcasts player movements.
- Ensures thread-safe client management using mutex locks.

## Installation & Setup

### Prerequisites
- Go installed on your system
- Gorilla WebSocket package

To install the required package, run:
```sh
go get -u github.com/gorilla/websocket
```

### Running the Server
Clone the repository and navigate to the project directory:
```sh
git clone <repo_url>
cd <project_directory>
```

Run the WebSocket server:
```sh
go run main.go
```

The server will start on port `4001` and listen for incoming WebSocket connections at:
```
ws://localhost:4001/ws
```

## WebSocket Events
The server processes and broadcasts the following events:

### 1. `newPlayer`
Sent when a new player joins the game.
#### Example Payload:
```json
{
  "event": "newPlayer",
  "player": {
    "name": "Player1",
    "id": "12345"
  }
}
```

### 2. `playerMovement`
Sent when a player moves their snake.
#### Example Payload:
```json
{
  "event": "playerMovement",
  "player": {
    "name": "Player1"
  },
  "key": "ArrowUp"
}
```

### 3. `playerDisconnected`
Broadcasted when a player disconnects.
#### Example Payload:
```json
{
  "event": "playerDisconnected",
  "id": "12345"
}
```

## Handling Disconnections
When a player disconnects, the server removes the client from the active list and notifies other players.

## Build commands

- `go build -o go-server`
- `sudo mv go-server /usr/local/bin`
- `sudo systemctl restart go-server`

## Service file path
- `/etc/systemd/system/go-server.service`