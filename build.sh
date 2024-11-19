#!/bin/bash

go build -gcflags="-e"

if [ "$1" == "debug" ]; then
    go build -gcflags="-e -l -N"
elif [ "$1" == "web" ]; then
	env GOOS=js GOARCH=wasm go build -gcflags="-e" -o web_build/minesweeper.wasm
elif [ "$1" == "" ]; then
    go build -gcflags="-e"
else
    echo invalid command $1
fi
