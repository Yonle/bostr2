package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/Yonle/bostr2/relayHandler"
)

func Accept_Websocket(w http.ResponseWriter, r *http.Request, ip string, ua string) {
	var AcceptOptions = &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionContextTakeover,
	}

	isApple := (strings.Contains(ua, "CFNetwork") || strings.Contains(ua, "Safari")) && !strings.Contains(ua, "Chrome") && !strings.Contains(ua, "Firefox")

	// compression is fucked on apple & nobody knows why.
	if isApple {
		AcceptOptions.CompressionMode = websocket.CompressionDisabled
	}

	conn, err := websocket.Accept(w, r, AcceptOptions)

	if err != nil {
		return
	}

	defer conn.CloseNow()

	log.Printf("%s connected (%s)", ip, ua)

	conn.SetReadLimit(-1)

	// we are not expecting normal users would remain connected for more than 30 minutes.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	defer cancel()

	var relaySession = relayHandler.NewSession(ctx)
	var s = Session{
		ClientIP: ip,

		ClientREQ:   make(MessageChan),
		ClientCLOSE: make(MessageChan),
		ClientEVENT: make(MessageChan),

		events:        make(SessionEvents),
		pendingEOSE:   make(SessionEOSEs),
		subscriptions: make(SessionSubs),

		destroyed: make(chan struct{}),

		relay: &relaySession,
		conn:  conn,
		ctx:   ctx,
	}

	s.StartListening()

	defer log.Printf("%s disconnect (%s)", ip, ua)

listener:
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			log.Printf("%s: %v", ip, err)
			break listener
		}

		var data []json.RawMessage

		if err := json.Unmarshal(msg, &data); err != nil {
			// doesn't looks right
			wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: your json doesn't looks right."})
			continue listener
		}

		if len(data) < 1 {
			wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: does not looks like there's something in your message."})
			continue listener
		}

		var cmd string
		if err := json.Unmarshal(data[0], &cmd); err != nil {
			wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: please check your command."})
			continue listener
		}

		switch cmd {
		case "REQ":
			if len(data) < 3 {
				wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: invalid REQ"})
				continue listener
			}

			s.ClientREQ <- data
		case "CLOSE":
			if len(data) < 2 {
				wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: invalid CLOSE"})
				continue listener
			}

			s.ClientCLOSE <- data
		case "EVENT":
			if len(data) < 2 {
				wsjson.Write(ctx, conn, [2]string{"NOTICE", "error: invalid EVENT"})
				continue listener
			}

			s.ClientEVENT <- data
		default:
			wsjson.Write(ctx, conn, [2]string{"NOTICE", fmt.Sprintf("error: unknown command %s", cmd)})
		}
	}
}
