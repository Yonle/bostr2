
all: bostr2

bostr2: main.go http.go websocket.go bouncer.go config.go
	go build -v -o bostr2 main.go http.go websocket.go bouncer.go config.go

install:
	go install -v .

dev: main.go http.go websocket.go bouncer.go config.go
	go run -v main.go http.go websocket.go bouncer.go config.go
