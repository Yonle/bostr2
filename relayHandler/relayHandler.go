package relayHandler

import (
	"context"
	"nhooyr.io/websocket"
)

func NewSession(ctx context.Context) RelaySession {
	return RelaySession{
		ctx: ctx,

		relays: make(SessionRelays),

		UpEVENT:     make(MessageChan),
		UpEOSE:      make(MessageChan),
		UpConnected: make(chan *websocket.Conn),
	}
}
