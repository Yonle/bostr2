
all: bostr2

bostr2: main.go http.go websocket.go bouncer.go config.go config.example.yaml ratelimiter.go killer.go
	go build -v -o bostr2 main.go http.go websocket.go bouncer.go config.go ratelimiter.go killer.go

install:
	go install -v .

dev: main.go http.go websocket.go bouncer.go config.go ratelimiter.go killer.go
	go run -v main.go http.go websocket.go bouncer.go config.go ratelimiter.go killer.go
