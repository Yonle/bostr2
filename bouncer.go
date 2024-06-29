package main

import (
	"context"
	"log"
//	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/Yonle/bostr2/relayHandler"
)

type SessionEvents map[string]map[string]struct{}
type SessionEOSEs map[string]int
type SessionSubs map[string][]interface{}

type MessageChan chan []interface{}

type Session struct {
	ClientIP string

	ClientREQ   MessageChan
	ClientCLOSE MessageChan
	ClientEVENT MessageChan

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
		<-s.ctx.Done()
		/*go func() {
			time.Sleep(7 * time.Second)
			select {
			case <-s.destroyed:
			default:
				panic("something hangs.")
			}
		}()*/
		s.relay.Wait()

		close(s.destroyed)

		log.Println(s.ClientIP, "=================== clean shutdown finished.")

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
			case conn, open := <-s.relay.UpConnected:
				if !open {
					continue listener
				}

				for subID, filters := range s.subscriptions {
					ReqData := append([]interface{}{"REQ", subID}, filters...)
					wsjson.Write(s.ctx, conn, ReqData)
				}

			case d, open := <-s.ClientREQ:
				if !open {
					continue listener
				}
				subID, ok1 := d[1].(string)
				if !ok1 {
					wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subID is not a string"})
					continue listener
				}

				filters := d[2:]

				s.subscriptions[subID] = filters
				s.events[subID] = make(map[string]struct{})
				s.pendingEOSE[subID] = 0

				s.relay.Broadcast(d)

			case d, open := <-s.ClientCLOSE:
				if !open {
					continue listener
				}
				subID, ok1 := d[1].(string)
				if !ok1 {
					wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subID is not a string"})
					continue listener
				}

				delete(s.subscriptions, subID)
				delete(s.events, subID)
				delete(s.pendingEOSE, subID)

				s.relay.Broadcast(d)

			case d, open := <-s.ClientEVENT:
				if !open {
					continue listener
				}
				event, ok1 := d[1].(map[string]interface{})
				if !ok1 {
					wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: invalid event"})
					continue listener
				}

				id, ok2 := event["id"].(string)
				if !ok2 {
					wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: invalid event"})
					continue listener
				}

				s.relay.Broadcast(d)

				wsjson.Write(s.ctx, s.conn, [4]interface{}{"OK", id, true, ""})
			case <-s.destroyed:
				break listener
			}
		}
	}()
}
