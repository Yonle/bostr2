package main

import (
	"context"
	"sync"
	//"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/Yonle/bostr2/relayHandler"
)

type SessionEvents map[string]map[string]struct{}
type SessionEOSEs map[string]int
type SessionSubs map[string][]interface{}

type MessageChan chan []interface{}

type ClientEvents []map[string]interface{}

type Session struct {
	ClientIP string

	ClientREQ    MessageChan
	ClientCLOSE  MessageChan
	ClientEVENT  MessageChan
	clientEvents ClientEvents

	events        SessionEvents
	pendingEOSE   SessionEOSEs
	subscriptions SessionSubs

	relay relayHandler.RelaySession

	destroyed chan struct{}
	once      sync.Once
	conn      *websocket.Conn
	ctx       context.Context
}

func (s *Session) Start() {
	s.relay.Init(config.Relays)
}

func (s *Session) StartListening() {
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

				s.handleUpstreamEVENT(d)
			case d, open := <-s.relay.UpEOSE:
				if !open {
					continue listener
				}

				s.handleUpstreamEOSE(d)
			case conn, open := <-s.relay.UpConnected:
				if !open {
					continue listener
				}

				s.resendEvents(conn)
				s.reopenSubscriptions(conn)

			case d, open := <-s.ClientREQ:
				if !open {
					continue listener
				}
				s.handleClientREQ(d)

			case d, open := <-s.ClientCLOSE:
				if !open {
					continue listener
				}
				s.handleClientCLOSE(d)

			case d, open := <-s.ClientEVENT:
				if !open {
					continue listener
				}
				s.handleClientEVENT(d)

			case <-s.destroyed:
				break listener
			}
		}
	}()
}

func (s *Session) reopenSubscriptions(conn *websocket.Conn) {
	for subID, filters := range s.subscriptions {
		data := append([]interface{}{"REQ", subID}, filters...)
		wsjson.Write(s.ctx, conn, data)
	}
}

func (s *Session) resendEvents(conn *websocket.Conn) {
	for _, event := range s.clientEvents {
		data := [2]interface{}{"EVENT", event}
		wsjson.Write(s.ctx, conn, data)
	}
}

func (s *Session) handleClientREQ(d []interface{}) {
	subID, ok1 := d[1].(string)
	if !ok1 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subID is not a string"})
		return
	}

	filters := d[2:]

	s.subscriptions[subID] = filters
	s.events[subID] = make(map[string]struct{})
	s.pendingEOSE[subID] = 0

	s.once.Do(s.Start)
	s.relay.Broadcast(d)
}

func (s *Session) handleClientCLOSE(d []interface{}) {
	subID, ok1 := d[1].(string)
	if !ok1 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subID is not a string"})
		return
	}

	delete(s.subscriptions, subID)
	delete(s.events, subID)
	delete(s.pendingEOSE, subID)

	s.relay.Broadcast(d)
	wsjson.Write(s.ctx, s.conn, [3]string{"CLOSED", subID, ""})
}

func (s *Session) handleClientEVENT(d []interface{}) {
	event, ok1 := d[1].(map[string]interface{})
	if !ok1 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: invalid EVENT"})
		return
	}

	id, ok2 := event["id"].(string)
	if !ok2 {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: invalid EVENT"})
		return
	}

	s.once.Do(s.Start)

	s.relay.Broadcast(d)
	s.clientEvents = append(s.clientEvents, event)

	wsjson.Write(s.ctx, s.conn, [4]interface{}{"OK", id, true, ""})
}

func (s *Session) handleUpstreamEVENT(d []interface{}) {
	subID, ok1 := d[1].(string)

	if !ok1 {
		return
	}

	if _, ok := s.events[subID]; !ok {
		return
	}

	event, ok2 := d[2].(map[string]interface{})
	if !ok2 {
		return
	}

	eventID, ok3 := event["id"].(string)

	if !ok3 {
		return
	}

	if _, ok := s.events[subID][eventID]; ok {
		return
	}

	s.events[subID][eventID] = struct{}{}
	wsjson.Write(s.ctx, s.conn, d)

	if _, ok := s.pendingEOSE[subID]; ok {
		if len(s.events[subID]) >= 500 {
			delete(s.pendingEOSE, subID)
			wsjson.Write(s.ctx, s.conn, [2]string{"EOSE", subID})
		}
	}
}

func (s *Session) handleUpstreamEOSE(d []interface{}) {
	subID, ok := d[1].(string)
	if !ok {
		return
	}

	if _, ok := s.pendingEOSE[subID]; !ok {
		return
	}

	s.pendingEOSE[subID]++

	if s.pendingEOSE[subID] >= s.relay.HowManyRelaysAreConnected {
		delete(s.pendingEOSE, subID)
		wsjson.Write(s.ctx, s.conn, [2]string{"EOSE", subID})
	}
}
