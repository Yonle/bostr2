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
	upCONNECTED chan *websocket.Conn
	UpMessage   MessageConn // to sent to client

	events       map[string]map[string]struct{}
	pendingEOSE  map[string]int
	subscriptons map[string]Message
	relays       map[*websocket.Conn]struct{}
	wg           sync.WaitGroup
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
				subID := (*d)[1].(string)

				if _, ok := s.events[subID]; !ok {
					continue listener
				}

				event := (*d)[2].(map[string]interface{})
				eventID := event["id"].(string)

				_, ok := s.events[subID][eventID]

				if ok {
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
				subID := (*d)[1].(string)
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
		for {
			select {
			case d := <-s.ClientREQ:
				
				s.pendingEOSE[subID]
			case d := <-s.ClientCLOSE:

			case d := <-s.ClientEVENT:
			}
		}
	}()

	// deal with messages like broadcast,
	// if not messages then deal with s.relays
	go func() {
		for {

		}
	}
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
