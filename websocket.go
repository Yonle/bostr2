package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/Yonle/bostr2/relayHandler"
)

var AcceptOptions = &websocket.AcceptOptions{
	InsecureSkipVerify: true,
	CompressionMode:    websocket.CompressionContextTakeover,
}

func Accept_Websocket(w http.ResponseWriter, r *http.Request, ip string, ua string) {
	conn, err := websocket.Accept(w, r, AcceptOptions)

	if err != nil {
		return
	}

	defer conn.CloseNow()

	log.Printf("%s connected (%s)", ip, ua)

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	var relaySession = relayHandler.NewSession(ctx)

	var once sync.Once

	var s = Session{
		ClientIP: ip,

		ClientREQ:   make(MessageChan),
		ClientCLOSE: make(MessageChan),
		ClientEVENT: make(MessageChan),

		events:        make(SessionEvents),
		pendingEOSE:   make(SessionEOSEs),
		subscriptions: make(SessionSubs),

		destroyed: make(chan struct{}),

		relay: relaySession,
		conn:  conn,
		ctx:   ctx,
	}

	s.StartListening()

	defer log.Printf("%s disconnect (%s)", ip, ua)

listener:
	for {
		var json []interface{}
		if err := wsjson.Read(ctx, conn, &json); err != nil {
			break
		}

		if len(json) < 1 {
			wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: does not looks like there's something in your message."})
			continue listener
		}

		cmd, ok := json[0].(string)
		if !ok {
			wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: please check your command."})
			continue listener
		}

		switch cmd {
		case "REQ":
			if len(json) < 3 {
				wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: invalid REQ"})
				continue listener
			}

			once.Do(s.Start)
			s.ClientREQ <- json
		case "CLOSE":
			if len(json) < 2 {
				wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: invalid CLOSE"})
				continue listener
			}

			s.ClientCLOSE <- json
		case "EVENT":
			if len(json) < 2 {
				wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: invalid EVENT"})
				continue listener
			}

			once.Do(s.Start)
			s.ClientEVENT <- json
		default:
			wsjson.Write(ctx, conn, [2]string{"NOTICE", fmt.Sprintf("error: unknown command %s", cmd)})
		}
	}
}
