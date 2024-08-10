package relayHandler

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type MessageChan chan []json.RawMessage
type SessionRelays map[*websocket.Conn]struct{}

type RelaySession struct {
	ctx context.Context
	wg  sync.WaitGroup
	mu  sync.RWMutex

	relays SessionRelays

	UpEVENT     MessageChan
	UpEOSE      MessageChan
	UpConnected chan *websocket.Conn

	HowManyRelaysAreConnected int
}

var DialOptions = &websocket.DialOptions{}

func (s *RelaySession) Init(r []string) {
	s.wg.Add(len(r))
	for _, u := range r {
		go s.connect(u)
	}
}

func (s *RelaySession) connect(url string) {
	defer s.wg.Done()

listener:
	for {
		dialCtx, dialCancel := context.WithTimeout(s.ctx, 5*time.Second)
		conn, _, err := websocket.Dial(dialCtx, url, DialOptions)
		dialCancel()

		select {
		case <-s.ctx.Done():
			if conn != nil {
				conn.CloseNow()
			}
			break listener
		default:
			if err != nil {
				time.Sleep(5 * time.Second)
				continue listener
			}
		}

		conn.SetReadLimit(-1)

		s.add(conn)
		s.HowManyRelaysAreConnected++

	messageListener:
		for {
			mt, msg, err := conn.Read(s.ctx)
			if err != nil {
				break messageListener
			}

			if mt != websocket.MessageText {
				break messageListener
			}

			var data []json.RawMessage
			if err := json.Unmarshal(msg, &data); err != nil {
				continue messageListener
			}

			var cmd string

			if err := json.Unmarshal(data[0], &cmd); err != nil {
				// ignore
				continue messageListener
			}

			switch cmd {
			case "EVENT":
				s.UpEVENT <- data
			case "EOSE":
				s.UpEOSE <- data
			}
		}

		s.del(conn)
		conn.CloseNow()
		s.HowManyRelaysAreConnected--

		select {
		case <-s.ctx.Done():
			break listener
		default:
			time.Sleep(5 * time.Second)
			continue listener
		}
	}
}

func (s *RelaySession) add(conn *websocket.Conn) {
	s.UpConnected <- conn

	defer s.mu.Unlock()
	s.mu.Lock()

	s.relays[conn] = struct{}{}
}

func (s *RelaySession) del(conn *websocket.Conn) {
	defer s.mu.Unlock()
	s.mu.Lock()

	delete(s.relays, conn)
}

func (s *RelaySession) Broadcast(data []json.RawMessage) {
	defer s.mu.RUnlock()
	s.mu.RLock()

	for conn := range s.relays {
		wsjson.Write(s.ctx, conn, data)
	}
}

func (s *RelaySession) Wait() {
	s.wg.Wait()
	close(s.UpEVENT)
	close(s.UpEOSE)
}
