
all: bostr2

bostr2: main.go websocket.go bouncer.go config.go
	go build -o bostr2 main.go websocket.go bouncer.go config.go

install:
	go install .

dev: main.go websocket.go bouncer.go config.go
	go run main.go websocket.go bouncer.go config.go
