#!/bin/bash

trap 'kill $(jobs -p)' EXIT


while true; do
    inotifywait -e modify -r ./*.go

    echo "Restarting local server..."
    pkill -f "go run ."
    go run . &

    echo "Building and restarting online server..."
    go build -o go-server
    sudo mv go-server /usr/local/bin
    sudo systemctl restart go-server

    echo "Both local and online servers restarted."
done
