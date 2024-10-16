@echo off

setlocal EnableDelayedExpansion

if "%1"=="debug" (
	go build -gcflags="-e -l -N"
	goto :quit
)

if "%1"=="web" (
	set "GOOS=js"
	set "GOARCH=wasm"
	go build -gcflags="-e" -o web_build\minesweeper.wasm
	goto :quit
)

go build -gcflags="-e"
goto :quit

:quit

endlocal
