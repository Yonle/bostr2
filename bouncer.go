package main

import (
	"context"
	"fmt"
	"sync"

	//"time"
	"encoding/json"
	"github.com/nbd-wtf/go-nostr"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"codeberg.org/Yonle/bostr2/relayHandler"
)

type SessionEvents map[string]map[string]struct{}
type SessionEOSEs map[string]int
type SessionSubs map[string]nostr.Filters

type MessageChan chan []json.RawMessage

type ClientEvents []nostr.Event

type Session struct {
	ClientIP string

	ClientREQ    MessageChan
	ClientCLOSE  MessageChan
	ClientEVENT  MessageChan
	clientEvents ClientEvents

	events        SessionEvents
	pendingEOSE   SessionEOSEs
	subscriptions SessionSubs

	relay *relayHandler.RelaySession

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
		data := []interface{}{"REQ", subID}
		for _, filter := range filters {
			data = append(data, filter)
		}

		wsjson.Write(s.ctx, conn, data)
	}
}

func (s *Session) resendEvents(conn *websocket.Conn) {
	for _, event := range s.clientEvents {
		data := []interface{}{"EVENT", event}

		wsjson.Write(s.ctx, conn, data)
	}
}

func (s *Session) handleClientREQ(d []json.RawMessage) {
	var subID string
	if err := json.Unmarshal(d[1], &subID); err != nil {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subID is not a string"})
		return
	}

	var filters nostr.Filters

	for index, rawFilter := range d[2:] {
		var filter nostr.Filter
		if err := filter.UnmarshalJSON(rawFilter); err != nil {
			// if it doesn't looks finje, then just stop.
			wsjson.Write(s.ctx, s.conn, [3]string{"CLOSED", subID, fmt.Sprintf("error: one of your filter on index %d does not looks right.", index)})
			return
		}

		filters = append(filters, filter)
	}

	s.subscriptions[subID] = filters
	s.events[subID] = make(map[string]struct{})
	s.pendingEOSE[subID] = 0

	s.once.Do(s.Start)
	s.relay.Broadcast(d)
}

func (s *Session) handleClientCLOSE(d []json.RawMessage) {
	var subID string
	if err := json.Unmarshal(d[1], &subID); err != nil {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: received subID is not a string"})
		return
	}

	delete(s.subscriptions, subID)
	delete(s.events, subID)
	delete(s.pendingEOSE, subID)

	s.relay.Broadcast(d)
	wsjson.Write(s.ctx, s.conn, [3]string{"CLOSED", subID, ""})
}

func (s *Session) handleClientEVENT(d []json.RawMessage) {
	var event nostr.Event
	if err := event.UnmarshalJSON(d[1]); err != nil {
		wsjson.Write(s.ctx, s.conn, [2]string{"NOTICE", "error: invalid EVENT"})
		return
	}

	id := event.GetID()

	s.clientEvents = append(s.clientEvents, event)

	s.once.Do(s.Start)
	s.relay.Broadcast(d)

	wsjson.Write(s.ctx, s.conn, [4]interface{}{"OK", id, true, ""})
}

func (s *Session) handleUpstreamEVENT(d []json.RawMessage) {
	var subID string
	if err := json.Unmarshal(d[1], &subID); err != nil {
		return
	}

	if _, ok := s.events[subID]; !ok {
		return
	}

	var event nostr.Event

	if err := event.UnmarshalJSON(d[2]); err != nil {
		return
	}

	eventID := event.GetID()

	if _, ok := s.events[subID][eventID]; ok {
		return
	}

	// get the filters and validate
	if filters := s.subscriptions[subID]; !filters.Match(&event) {
		// "shut"
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

func (s *Session) handleUpstreamEOSE(d []json.RawMessage) {
	var subID string
	if err := json.Unmarshal(d[1], &subID); err != nil {
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
