package main

import (
	"context"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Message interface{}
type MessageChan chan *[]Message

type Session struct {
	ClientIP string

	// these three Client* channels are for receiving messages from client
	ClientREQ     MessageChan
	ClientCLOSE   MessageChan
	ClientEVENT   MessageChan
	clientMessage MessageChan // to sent to upstreams

	// these two up* channels are for receiving messages from upstream relays
	upEVENT MessageChan
	upEOSE  MessageChan

	// if this channel received the websocket.Conn, Put to s.relays
	upAdd     chan *websocket.Conn
	upDel     chan *websocket.Conn
	UpMessage MessageChan // to sent to client

	events        map[string]map[string]struct{}
	pendingEOSE   map[string]int
	subscriptions map[string]*[]Message
	relays        map[*websocket.Conn]struct{}
	wg            sync.WaitGroup
}

func (s *Session) Start() {
	// connect first
	for _, url := range config.Relays {
		s.wg.Add(1)
		go s.newConn(url)
	}

	// receive stuff from upstream
	go func() {
	listener:
		for {
			select {
			case d := <-s.upEVENT:
				subID, ok1 := (*d)[1].(string)

				if !ok1 {
					continue listener
				}

				if _, ok := s.events[subID]; !ok {
					continue listener
				}

				event, ok2 := (*d)[2].(map[string]interface{})
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
				s.UpMessage <- d

				if _, ok := s.pendingEOSE[subID]; ok {
					if len(s.events[subID]) >= 500 {
						delete(s.pendingEOSE, subID)
						s.UpMessage <- &[]Message{"EOSE", subID}
					}
				}
			case d := <-s.upEOSE:
				subID, ok := (*d)[1].(string)
				if !ok {
					// unusual.
					continue listener
				}

				if _, ok := s.pendingEOSE[subID]; !ok {
					continue listener
				}

				s.pendingEOSE[subID]++

				if s.pendingEOSE[subID] >= len(s.relays) {
					delete(s.pendingEOSE, subID)
					s.UpMessage <- d
				}
			}
		}
	}()

	// receive stuff from client
	go func() {
	listener:
		for {
			select {
			case d := <-s.ClientREQ:
				subID, ok := (*d)[1].(string)
				if !ok {
					s.UpMessage <- &[]Message{"NOTICE", "error: invalid REQ"}
					continue listener
				}

				filters := (*d)[2:]
				s.subscriptions[subID] = &filters
				s.events[subID] = make(map[string]struct{})
				s.pendingEOSE[subID] = 0

				s.clientMessage <- d
			case d := <-s.ClientCLOSE:
				subID, ok := (*d)[1].(string)
				if !ok {
					s.UpMessage <- &[]Message{"NOTICE", "error: invalid EVENT"}
					continue listener
				}

				delete(s.subscriptions, subID)
				delete(s.events, subID)
				delete(s.pendingEOSE, subID)

				s.clientMessage <- d
			case d := <-s.ClientEVENT:
				event, ok := (*d)[1].(map[string]interface{})
				if !ok {
					s.UpMessage <- &[]Message{"NOTICE", "error: invalid EVENT"}
					continue listener
				}

				id, actuallyOk := event["id"]
				if !actuallyOk {
					s.UpMessage <- &[]Message{"NOTICE", "error: invalid EVENT"}
					continue listener
				}

				s.clientMessage <- d
				s.UpMessage <- &[]Message{"OK", id, true, ""}
			}
		}
	}()

	// deal with messages like broadcast,
	// if not messages then deal with s.relays
	go func() {
		for {
			select {
			// deal with relays
			// add websocket.Conn to s.relays
			// or delete websocket.Conn on s.relays
			case conn := <-s.upAdd:
				s.relays[conn] = struct{}{}
			case conn := <-s.upDel:
				delete(s.relays, conn)

			// deal with client message
			// broadcast validated Message to every single s.relays
			case d := <-s.clientMessage:
				for conn := range s.relays {
					ctx := context.Background()
					if err := wsjson.Write(ctx, conn, d); err != nil {
						s.upDel <- conn
					}
				}
			}
		}
	}()
}

func (s *Session) Destroy() {
	s.wg.Wait()
}

func (s *Session) newConn(url string) {
	defer s.wg.Done()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

		conn, _, err := websocket.Dial(ctx, url, nil)

		if err != nil {
			cancel()
			continue
		}

		rctx := context.Background()

		for {
			var json []Message
			if err := wsjson.Read(rctx, conn, &json); err != nil {
				break
			}

			switch json[0].(string) {
			case "EVENT":
				s.upEVENT <- &json
			case "EOSE":
				s.upEOSE <- &json
			}
		}

		cancel()
	}
}
