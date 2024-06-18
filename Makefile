
all: blyat

blyat: main.go websocket.go bouncer.go config.go
	go build -o blyat main.go websocket.go bouncer.go config.go

install:
	mv blyat ~/go/bin/

dev: main.go websocket.go bouncer.go config.go
	go run main.go websocket.go bouncer.go config.go
