package main

import (
  "log"
  "time"
  "sync"
  "github.com/gorilla/websocket"
)

type SessionSubIDs map[string]*[]interface{}
type SessionEventIDs map[string]map[string]struct{}
type SessionPendingEOSE map[string]int
type SessionRelays map[*websocket.Conn]struct{}

type Session struct {
  Owner *websocket.Conn
  Sub_IDs SessionSubIDs
  Event_IDs SessionEventIDs
  PendingEOSE SessionPendingEOSE
  Relays SessionRelays
  ready bool
  destroyed bool

  ownerWriteMu sync.RWMutex
  eventMu sync.Mutex
  eoseMu sync.Mutex
  relaysMu sync.Mutex
  connWriteMu sync.RWMutex
}

var dialer = websocket.Dialer{
  EnableCompression: true
}

func (s *Session) Exist() bool {
  return s.Owner != nil
}

func (s *Session) NewConn(url string) {
  if s.destroyed {
    return
  }

  conn, resp, err := dialer.Dial(url, nil)

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
    log.Printf("Получил неожиданный код статуса от %s (%s). Больше не подключаюсь.\n", url, resp.StatusCode)
    return
  }

  s.relaysMu.Lock()
  s.Relays[conn] = struct{}{}
  s.relaysMu.Unlock()

  log.Printf("%s присоединился к нам.\n", url)

  s.OpenSubscriptions(conn)

  var stop bool = false

  for {
    var data []interface{}
    if err := conn.ReadJSON(&data); err != nil {
      return
    }

    switch data[0].(string) {
    case "EVENT":
      s.HandleUpstreamEVENT(data, &stop)
    case "EOSE":
      s.HandleUpstreamEOSE(data, &stop)
    }

    if stop {
      return
    }
  }

  conn.Close()

  if !stop {
    s.Reconnect(conn, &url)
  } else {
    log.Printf("%s: Отключение\n", url)
  }
}

func (s *Session) Reconnect(conn *websocket.Conn, url *string) {
  log.Printf("Произошла ошибка при подключении к %s. Повторная попытка через 5 секунд....\n", *url);

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
      return;
    }
    go s.NewConn(url);
  }
}

func (s *Session) Broadcast(data *[]interface{}) {
  s.connWriteMu.Lock()
  defer s.connWriteMu.Unlock()

  for relay, _ := range s.Relays {
    relay.WriteJSON(data)
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

  return ok
}

func (s *Session) HandleUpstreamEVENT(data []interface{}, stop *bool) {
  if _, ok := s.Sub_IDs[data[1].(string)]; !ok {
    return
  }

  if event := data[2].(map[string]interface{}); s.HasEvent(data[1].(string), event["id"].(string)) {
    return
  }

  if err := s.WriteJSON(&data); err != nil {
    *stop = true
    return
  }
}

func (s *Session) HandleUpstreamEOSE(data []interface{}, stop *bool) {
  s.eoseMu.Lock()
  defer s.eoseMu.Unlock()

  if _, ok := s.PendingEOSE[data[1].(string)]; !ok {
    return
  }

  s.PendingEOSE[data[1].(string)]++
  if s.PendingEOSE[data[1].(string)] >= len(config.Relays) {
    delete(s.PendingEOSE, data[1].(string))
    if err := s.WriteJSON(&data); err != nil {
      *stop = true
      return
    }
  }
}

/*
func (s *Session) CountEvents(subid string) int {
  return len(s.Event_IDs[subid])
}
*/

func (s *Session) WriteJSON(data *[]interface{}) error {
  s.ownerWriteMu.Lock()
  defer s.ownerWriteMu.Unlock()

  return s.Owner.WriteJSON(data)
}

func (s *Session) OpenSubscriptions(conn *websocket.Conn) {
  s.connWriteMu.Lock()
  defer s.connWriteMu.Unlock()

  for id, filters := range s.Sub_IDs {
    ReqData := []interface{}{"REQ", id}
    ReqData = append(ReqData, *filters...)

    conn.WriteJSON(&ReqData)
  }
}

func (s *Session) Destroy(_ int, _ string) error {
  s.destroyed = true

  for relay, _ := range s.Relays {
    relay.Close()
  }

  return nil
}

func (s *Session) REQ(data *[]interface{}) {
  if !s.ready {
    s.StartConnect()
    s.ready = true
  }

  subid := (*data)[1].(string)
  filters := (*data)[2:]

  s.CLOSE(data, false)
  s.Event_IDs[subid] = make(map[string]struct{})
  s.PendingEOSE[subid] = 0
  s.Sub_IDs[subid] = &filters;
}

func (s *Session) CLOSE(data *[]interface{}, sendClosed bool) {
  subid := (*data)[1].(string)

  delete(s.Event_IDs, subid)
  delete(s.Sub_IDs, subid)
  delete(s.PendingEOSE, subid)

  if sendClosed {
    s.WriteJSON(&[]interface{}{"CLOSED", subid, ""})
  }
}

func (s *Session) EVENT(data *[]interface{}) bool {
  if !s.ready {
    s.StartConnect()
    s.ready = true
  }

  event := (*data)[1].(map[string]interface{})
  id, ok := event["id"]
  if ok {
    s.WriteJSON(&[]interface{}{"OK", id, true, ""})
  }

  return ok
}
