package relayHandler

import (
	"context"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
	"sync"
	"time"
)

type MessageChan chan []interface{}
type SessionRelays map[*websocket.Conn]struct{}

type RelaySession struct {
	ctx context.Context
	wg  sync.WaitGroup
	mu  sync.RWMutex

	relays SessionRelays

	UpEVENT MessageChan
	UpEOSE  MessageChan

	HowManyRelaysAreConnected	int
}

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
		conn, _, err := websocket.Dial(dialCtx, url, nil)
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

		s.add(conn)
		s.HowManyRelaysAreConnected++

	messageListener:
		for {
			var json []interface{}
			if err := wsjson.Read(s.ctx, conn, &json); err != nil {
				break messageListener
			}

			// big failure for next gen
			switch json[0].(string) {
			case "EVENT":
				s.UpEVENT <- json
			case "EOSE":
				s.UpEOSE <- json
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
	defer s.mu.Unlock()
	s.mu.Lock()

	s.relays[conn] = struct{}{}
}

func (s *RelaySession) del(conn *websocket.Conn) {
	defer s.mu.Unlock()
	s.mu.Lock()

	delete(s.relays, conn)
}

func (s *RelaySession) Broadcast(data []interface{}) {
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
