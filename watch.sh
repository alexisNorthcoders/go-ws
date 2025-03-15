#!/bin/bash

trap 'kill $(jobs -p)' EXIT

restart_servers() {
    echo "Restarting local server..."
    pkill -f "/tmp/go-build.*"
    go run . &

    echo "Building and restarting online server..."
    go build -o go-server
    sudo mv go-server /usr/local/bin
    sudo systemctl restart go-server

    echo "Both local and online servers restarted."
}

restart_servers

while true; do
    inotifywait -e modify -r ./*.go
    restart_servers
done
