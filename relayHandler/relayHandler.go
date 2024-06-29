package relayHandler

import (
	"context"
)

func NewSession(ctx context.Context) RelaySession {
	return RelaySession{
		ctx: ctx,

		relays: make(SessionRelays),

		UpEVENT: make(MessageChan),
		UpEOSE:  make(MessageChan),
	}
}
