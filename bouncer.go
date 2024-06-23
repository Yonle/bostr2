package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type SessionSubIDs map[string]*[]interface{}
type SessionEventIDs map[string]map[string]struct{}
type SessionPendingEOSE map[string]int
type SessionRelays map[*websocket.Conn]struct{}
type SessionUpstreamMessage chan *[]byte
type SessionDoneChannel chan struct{}

type Session struct {
	Owner           *websocket.Conn
	Sub_IDs         SessionSubIDs
	Event_IDs       SessionEventIDs
	PendingEOSE     SessionPendingEOSE
	Relays          SessionRelays
	UpstreamMessage SessionUpstreamMessage
	Done            SessionDoneChannel
	ready           bool
	destroyed       bool

	eventMu     sync.Mutex
	eoseMu      sync.Mutex
	relaysMu    sync.Mutex
	connWriteMu sync.Mutex
	subMu       sync.Mutex
}

var dialer = websocket.Dialer{}

func (s *Session) Exist() bool {
	return s.Owner != nil
}

func (s *Session) NewConn(url string) {
	if s.destroyed {
		return
	}

	connHeaders := make(http.Header)
	connHeaders.Add("User-Agent", "Blyat; Nostr relay bouncer; https://github.com/Yonle/blyat")

	conn, resp, err := dialer.Dial(url, connHeaders)

	if s.destroyed && conn != nil {
		conn.Close()
		return
	}

	if err != nil && !s.destroyed {
		s.Reconnect(conn, &url)
		return
	}

	if s.destroyed {
		if conn != nil {
			conn.Close()
		}
		return
	}

	if resp.StatusCode >= 500 {
		s.Reconnect(conn, &url)
		return
	} else if resp.StatusCode > 101 {
		log.Printf("Получил неожиданный код статуса от %s (%d). Больше не подключаюсь.\n", url, resp.StatusCode)
		return
	}

	s.relaysMu.Lock()
	s.Relays[conn] = struct{}{}
	s.relaysMu.Unlock()

	log.Printf("%s присоединился к нам.\n", url)

	s.OpenSubscriptions(conn)

	defer s.Reconnect(conn, &url)
	defer conn.Close()

	for {
		var data []interface{}
		if err := conn.ReadJSON(&data); err != nil {
			return
		}

		if data == nil {
			return
		}

		switch data[0].(string) {
		case "EVENT":
			s.HandleUpstreamEVENT(data)
		case "EOSE":
			s.HandleUpstreamEOSE(data)
		}
	}
}

func (s *Session) Reconnect(conn *websocket.Conn, url *string) {
	if s.destroyed {
		log.Printf("%s: Отключение\n", *url)
		return
	}

	log.Printf("Произошла ошибка при подключении к %s. Повторная попытка через 5 секунд....\n", *url)

	s.relaysMu.Lock()
	delete(s.Relays, conn)
	s.relaysMu.Unlock()

	time.Sleep(5 * time.Second)
	if s.destroyed {
		return
	}
	go s.NewConn(*url)
}

func (s *Session) StartConnect() {
	for _, url := range config.Relays {
		if s.destroyed {
			return
		}
		go s.NewConn(url)
	}
}

func (s *Session) Broadcast(data *[]interface{}) {
	JsonData, _ := json.Marshal(*data)

	s.relaysMu.Lock()
	defer s.relaysMu.Unlock()

	for relay := range s.Relays {
		s.connWriteMu.Lock()
		relay.WriteMessage(websocket.TextMessage, JsonData)
		s.connWriteMu.Unlock()
	}
}

func (s *Session) HasEvent(subid string, event_id string) bool {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	events := s.Event_IDs[subid]
	if events == nil {
		return true
	}

	_, ok := events[event_id]

	if !ok {
		events[event_id] = struct{}{}
	}

	if len(events) > 500 {
		s.eoseMu.Lock()
		if _, ok := s.PendingEOSE[subid]; ok {
			delete(s.PendingEOSE, subid)
			s.WriteJSON(&[]interface{}{"EOSE", subid})
		}
		s.eoseMu.Unlock()
	}

	return ok
}

func (s *Session) HandleUpstreamEVENT(data []interface{}) {
	if len(data) < 3 {
		return
	}

	s.subMu.Lock()
	if _, ok := s.Sub_IDs[data[1].(string)]; !ok {
		s.subMu.Unlock()
		return
	}
	s.subMu.Unlock()

	if event := data[2].(map[string]interface{}); s.HasEvent(data[1].(string), event["id"].(string)) {
		return
	}

	s.WriteJSON(&data)
}

func (s *Session) HandleUpstreamEOSE(data []interface{}) {
	if len(data) < 2 {
		return
	}

	s.eoseMu.Lock()
	defer s.eoseMu.Unlock()

	if _, ok := s.PendingEOSE[data[1].(string)]; !ok {
		return
	}

	s.PendingEOSE[data[1].(string)]++
	if s.PendingEOSE[data[1].(string)] >= len(config.Relays) {
		delete(s.PendingEOSE, data[1].(string))
		s.WriteJSON(&data)
	}
}

/*
func (s *Session) CountEvents(subid string) int {
  return len(s.Event_IDs[subid])
}
*/

func (s *Session) WriteJSON(data *[]interface{}) {
	defer func() {
		if recover() != nil {
			return
		}
	}()

	JsonData, _ := json.Marshal(*data)

	s.UpstreamMessage <- &JsonData
}

func (s *Session) OpenSubscriptions(conn *websocket.Conn) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	for id, filters := range s.Sub_IDs {
		ReqData := []interface{}{"REQ", id}
		ReqData = append(ReqData, *filters...)
		JsonData, _ := json.Marshal(ReqData)

		s.connWriteMu.Lock()
		conn.WriteMessage(websocket.TextMessage, JsonData)
		s.connWriteMu.Unlock()
	}
}

func (s *Session) Destroy() {
	s.destroyed = true

	s.relaysMu.Lock()
	for relay := range s.Relays {
		go relay.Close()
	}
	s.relaysMu.Unlock()

	s.Done <- struct{}{}
}

func (s *Session) REQ(data *[]interface{}) {
	if !s.ready {
		s.StartConnect()
		s.ready = true
	}

	subid := (*data)[1].(string)
	filters := (*data)[2:]

	s.CLOSE(data, false)

	s.eventMu.Lock()
	s.Event_IDs[subid] = make(map[string]struct{})
	s.eventMu.Unlock()

	s.eoseMu.Lock()
	s.PendingEOSE[subid] = 0
	s.eoseMu.Unlock()

	s.subMu.Lock()
	s.Sub_IDs[subid] = &filters
	s.subMu.Unlock()

	s.Broadcast(data)
}

func (s *Session) CLOSE(data *[]interface{}, sendClosed bool) {
	subid := (*data)[1].(string)

	s.eventMu.Lock()
	delete(s.Event_IDs, subid)
	s.eventMu.Unlock()

	s.subMu.Lock()
	delete(s.Sub_IDs, subid)
	s.subMu.Unlock()

	s.eoseMu.Lock()
	delete(s.PendingEOSE, subid)
	s.eoseMu.Unlock()

	if sendClosed {
		s.WriteJSON(&[]interface{}{"CLOSED", subid, ""})
	}

	s.Broadcast(data)
}

func (s *Session) EVENT(data *[]interface{}) {
	if !s.ready {
		s.StartConnect()
		s.ready = true
	}

	event := (*data)[1].(map[string]interface{})
	id, ok := event["id"]
	if !ok {
		s.WriteJSON(&[]interface{}{"NOTICE", "Неверный объект."})
		return
	}

	s.WriteJSON(&[]interface{}{"OK", id, true, ""})
	s.Broadcast(data)
}
