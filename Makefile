
all: bostr2

bostr2: http.go websocket.go bouncer.go config.go
	go build -o bostr2 http.go websocket.go bouncer.go config.go

install:
	go install .

dev: http.go websocket.go bouncer.go config.go
	go run http.go websocket.go bouncer.go config.go
