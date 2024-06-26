package main

import (
	"context"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Message interface{}
type MessageChan chan *[]Message
type WebSocketChan chan *websocket.Conn

type SessionEvents map[string]map[string]struct{}
type SessionEOSEs map[string]int
type SessionSubs map[string]*[]Message
type SessionRelays map[*websocket.Conn]struct{}

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
	upAdd     WebSocketChan
	upDel     WebSocketChan
	UpMessage MessageChan // to sent to client

	events        SessionEvents
	pendingEOSE   SessionEOSEs
	subscriptions SessionSubs
	relays        SessionRelays

	wg sync.WaitGroup

	destroyed chan struct{}
	ctx       context.Context
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

			// receive stuff from client

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
			case <-s.destroyed:
				break listener
			}
		}
	}()

	// deal with relays
	go func() {
	listener:
		for {
			select {
			case conn := <-s.upAdd:
				// add websocket.Conn to s.relays
				// or delete websocket.Conn on s.relays
				if s.isDestroyed() {
					if conn != nil {
						conn.CloseNow()
					}
					continue listener
				}

				s.relays[conn] = struct{}{}

				for subID, filters := range s.subscriptions {
					ReqData := append([]Message{"REQ", subID}, (*filters)...)
					wsjson.Write(s.ctx, conn, &ReqData)
				}

			case conn := <-s.upDel:
				delete(s.relays, conn)

			case d := <-s.clientMessage:
				// deal with client message
				// broadcast validated Message to every single s.relays
				for conn := range s.relays {
					wsjson.Write(s.ctx, conn, d)
				}

			case <-s.destroyed:
				break listener
			}
		}
	}()

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
						log.Println(s.ClientIP, "== SESSION SHUTDOWM HANG!! ==")
					}
				}()
				s.wg.Wait()

				close(s.destroyed)
				/*close(s.ClientREQ)
				close(s.ClientCLOSE)
				close(s.ClientEVENT)
				close(s.clientMessage)

				close(s.upEVENT)
				close(s.upEOSE)
				close(s.upAdd)
				close(s.upDel)
				close(s.UpMessage)*/

				log.Println(s.ClientIP, "=================== clean shutdown finished.")
				break listener
			}
		}
	}()
}

func (s *Session) isDestroyed() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func (s *Session) newConn(url string) {
	defer s.wg.Done()

listener:
	for {
		dialCtx, dialCancel := context.WithTimeout(s.ctx, 20*time.Second)
		conn, _, err := websocket.Dial(dialCtx, url, nil)
		dialCancel()

		if err != nil {
			select {
			case <-s.ctx.Done():
				break listener
			default:
				time.Sleep(5 * time.Second)
				continue listener
			}
		}

		s.upAdd <- conn

	messageListener:
		for {
			var json []Message
			if err := wsjson.Read(s.ctx, conn, &json); err != nil {
				break messageListener
			}

			switch json[0].(string) {
			case "EVENT":
				s.upEVENT <- &json
			case "EOSE":
				s.upEOSE <- &json
			}
		}

		conn.CloseNow()

		select {
		case <-s.ctx.Done():
			break listener
		default:
			s.upDel <- conn
			time.Sleep(5 * time.Second)
			continue listener
		}
	}
}
