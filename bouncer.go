package main

import (
	"context"
	"log"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/Yonle/bostr2/relayHandler"
)

type SessionEvents map[string]map[string]struct{}
type SessionEOSEs map[string]int
type SessionSubs map[string][]interface{}

type Session struct {
	ClientIP string

	events        SessionEvents
	pendingEOSE   SessionEOSEs
	subscriptions SessionSubs

	relay relayHandler.RelaySession

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
	go func() {
	listener:
		for {
			select {
			case d, open := <-s.relay.UpEVENT:
				if !open {
					continue listener
				}

				subID, ok1 := d[1].(string)

				if !ok1 {
					continue listener
				}

				if _, ok := s.events[subID]; !ok {
					continue listener
				}

				event, ok2 := d[2].(map[string]interface{})
				if !ok2 {
					continue listener
				}

				eventID, ok3 := event["id"].(string)

				if !ok3 {
					continue listener
				}

				if _, ok := s.events[subID][eventID]; ok {
					continue listener
				}

				s.events[subID][eventID] = struct{}{}
				wsjson.Write(s.ctx, s.conn, d)

				if _, ok := s.pendingEOSE[subID]; ok {
					if len(s.events[subID]) >= 500 {
						delete(s.pendingEOSE, subID)
						wsjson.Write(s.ctx, s.conn, [2]string{"EOSE", subID})
					}
				}
			case d, open := <-s.relay.UpEOSE:
				if !open {
					continue listener
				}

				subID, ok := d[1].(string)
				if !ok {
					continue listener
				}

				if _, ok := s.pendingEOSE[subID]; !ok {
					continue listener
				}

				s.pendingEOSE[subID]++

				if s.pendingEOSE[subID] >= s.relay.HowManyRelaysAreConnected {
					delete(s.pendingEOSE, subID)
					wsjson.Write(s.ctx, s.conn, [2]string{"EOSE", subID})
				}

			case <-s.destroyed:
				break listener
			}
		}
	}()
}

func (s *Session) REQ(data []interface{}) {
	subid, ok1 := data[1].(string)
	if !ok1 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subid is not a string"})
		return
	}

	filters := data[2:]

	s.subscriptions[subid] = filters
	s.events[subid] = make(map[string]struct{})
	s.pendingEOSE[subid] = 0

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

	wsjson.Write(s.ctx, s.conn, [4]interface{}{"OK", id, true, ""})
}
