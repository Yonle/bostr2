package pingpong

import (
	"context"
	"github.com/coder/websocket"
	"time"
)

// must run with goroutine
func Stare(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
lurker:
	for {
		select {
		case <-ctx.Done():
			break lurker
		case <-time.After(time.Second * 10):
			if err := conn.Ping(ctx); err != nil {
				cancel()
			}
		}
	}
}
