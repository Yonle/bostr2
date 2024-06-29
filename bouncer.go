package main

import (
	"context"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/Yonle/bostr2/relayHandler"
)

type SessionEvents map[string]map[string]struct{}
type SessionEOSEs map[string]int
type SessionSubs map[string]*[]interface{}

type Session struct {
	ClientIP string

	events        SessionEvents
	pendingEOSE   SessionEOSEs
	subscriptions SessionSubs

	relay      relayHandler.RelaySession
	listenerWg sync.WaitGroup

	destroyed chan struct{}
	conn      *websocket.Conn
	ctx       context.Context
}

func (s *Session) Start() {
	s.relay.Init(config.Relays)

	// deal with destroy request.
	go func() {
	listener:
		for {
			select {
			case <-s.ctx.Done():
				go func() {
					time.Sleep(5 * time.Second)
					select {
					case <-s.destroyed:
					default:
						panic("something hangs.")
					}
				}()
				s.relay.Wait()

				close(s.destroyed)

				log.Println(s.ClientIP, "=================== clean shutdown finished.")
				break listener
			}
		}
	}()

	// deal with what upstream says
	
}

func (s *Session) REQ(data []interface{}) {
	subid, ok1 := data[1].(string)
	if !ok1 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subid is not a string"})
		return
	}

	filters := data[2:]

	s.subscriptions[subid] := filters
	s.events := make(map[string]struct{})
	s.pendingEOSE := 0

	s.relay.Broadcast(data)
}

func (s *Session) CLOSE(data []interface{}) {
	subid, ok1 := data[1].(string)
	if !ok1 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subid is not a string"})
		return
	}

	delete(s.subscriptions, subid)
	delete(s.events, subid)
	delete(s.pendingEOSE, subid)

	s.relay.Broadcast(data)
}

func (s *Session) EVENT(data []interface{}) {
	event, ok1 := data[1].(map[string]interface{})
	if !ok1 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: invalid event"})
		return
	}

	id, ok2 := event["id"].(string)
	if !ok2 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: invalid event"})
		return
	}

	s.relay.Broadcast(data)

	wsjson.Write(s.ctx, s.conn, [4]string{"OK", id, true, ""})
}
