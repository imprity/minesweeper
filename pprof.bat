@echo off

if "%1"=="/?" (
	goto :display_help
)

if "%1"=="/h" (
	goto :display_help
)

if "%1"=="-h" (
	goto :display_help
)

if "%1"=="--help" (
	goto :display_help
)

if "%1"=="" (
	goto :display_help
)

if "%1"=="trace" (
	if "%2"=="" (
		curl -o trace.out http://localhost:6060/debug/pprof/trace
		go tool trace trace.out
		goto :quit
	) else (
		curl -o trace.out http://localhost:6060/debug/pprof/trace?seconds=%2
		go tool trace trace.out
		goto :quit
	)
)

if "%2"=="" (
	go tool pprof -http=:6969 minesweeper.exe http://localhost:6060/debug/pprof/%1
) else (
	go tool pprof -http=:6969 minesweeper.exe http://localhost:6060/debug/pprof/%1?seconds=%2
)

goto :quit

:display_help
echo batch file for pprof
echo usage pprof to-profile time
echo -
echo things to profile
echo     allocs
echo     block
echo     cmdline
echo     goroutine
echo     heap
echo     mutex
echo     profile
echo     threadcreate
echo     trace

:quit
