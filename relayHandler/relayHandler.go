package relayHandler

import (
	"context"
	"github.com/coder/websocket"
)

func NewSession(ctx context.Context, ip string) RelaySession {
	return RelaySession{
		ctx:      ctx,
		clientIP: ip,

		relays: make(SessionRelays),

		UpEVENT:     make(MessageChan),
		UpEOSE:      make(MessageChan),
		UpConnected: make(chan *websocket.Conn),
	}
}
