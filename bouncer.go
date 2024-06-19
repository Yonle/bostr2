package main

import (
  "log"
  "time"
  "sync"
  "github.com/gorilla/websocket"
)

type SessionSubIDs map[string]struct{}
type SessionEventIDs map[string]map[string]struct{}
type SessionPendingEOSE map[string]int

type Session struct {
  Owner *websocket.Conn
  Sub_IDs SessionSubIDs
  Event_IDs SessionEventIDs
  PendingEOSE SessionPendingEOSE
  Relays []*websocket.Conn
  mu sync.Mutex
}

var dialer = websocket.Dialer{}

func (s *Session) Delete(index int) {
  if index == 0 {
    return
  }

  s.Relays[index] = s.Relays[len(s.Relays)-1]
  s.Relays = s.Relays[:len(s.Relays)-1]
}

func (s *Session) Exist() bool {
  return s.Owner != nil
}

func (s *Session) NewConn(url string) {
  var index int = 0

  conn, resp, err := dialer.Dial(url, nil)

  if !s.Exist() {
    if err == nil {
      conn.Close()
    }
    return
  }

  if err != nil {
    log.Printf("Произошла ошибка при подключении к %s. Повторная попытка через 5 секунд....\n", url);

    s.Delete(index)
    go func() {
      time.Sleep(5 * time.Second)
      if !s.Exist() {
        return
      }

      s.NewConn(url)
    }()
    return
  }

  if resp.StatusCode >= 500 {
    s.Delete(index)

    go func() {
      time.Sleep(5 * time.Second)
      if !s.Exist() {
        return
      }

      s.NewConn(url)
    }()
    return
  } else if resp.StatusCode > 101 {
    log.Printf("Получил неожиданный код статуса от %s (%s). Больше не подключаюсь.\n", url, resp.StatusCode)
    return
  }

  index = len(s.Relays)

  s.Relays = append(s.Relays, conn)

  log.Printf("%s присоединился к нам.\n", url)

  go func() {
    var stop bool = false
    defer conn.Close()

    for {
      var data []interface{}
      if err := conn.ReadJSON(&data); err != nil {
        log.Printf("%s: Подделано!! Отключение\n", url)
        break
      }

      switch data[0].(string) {
      case "EVENT":
        if _, ok := s.Sub_IDs[data[1].(string)]; !ok {
          continue
        }
        if event := data[2].(map[string]interface{}); s.HasEvent(data[1].(string), event["id"].(string)) {
          continue
        }

        if err := s.WriteJSON(&data); err != nil {
          stop = true
          break
        }

      case "EOSE":
        if _, ok := s.Sub_IDs[data[1].(string)]; !ok {
          continue
        }

        if _, ok := s.PendingEOSE[data[1].(string)]; !ok {
          continue
        }

        s.PendingEOSE[data[1].(string)]++
        if s.PendingEOSE[data[1].(string)] >= len(config.Relays) {
          delete(s.PendingEOSE, data[1].(string))
          if err := s.WriteJSON(&data); err != nil {
            stop = true
            break
          }
          continue
        }
      }

      if stop {
        break
      }
    }
  }()
}

func (s *Session) StartConnect() {
  for _, url := range config.Relays {
    if !s.Exist() {
      break;
    }
    go s.NewConn(url);
  }
}

func (s *Session) Broadcast(data *[]interface{}) {
  for _, relay := range s.Relays {
    relay.WriteJSON(data)
  }
}

func (s *Session) HasEvent(subid string, event_id string) bool {
  // todo: do stuff
  events := s.Event_IDs[subid]
  _, ok := events[event_id]

  if !ok {
    events[event_id] = struct{}{}
  }

  return ok
}

func (s *Session) WriteJSON(data *[]interface{}) error {
  s.mu.Lock()
  defer s.mu.Unlock()
  return s.Owner.WriteJSON(data)
}

func (s *Session) REQ(data *[]interface{}) {
  subid := (*data)[1].(string)

  s.CLOSE(data, false)
  s.Event_IDs[subid] = make(map[string]struct{})
  s.PendingEOSE[subid] = 0
  s.Sub_IDs[subid] = struct{}{};
}

func (s *Session) CLOSE(data *[]interface{}, sendClosed bool) {
  subid := (*data)[1].(string)

  if _, ok := s.Event_IDs[subid]; ok {
    delete(s.Event_IDs, subid)
  }

  if _, ok := s.Sub_IDs[subid]; ok {
    delete(s.Sub_IDs, subid)
  }

  if _, ok := s.PendingEOSE[subid]; ok {
    delete(s.PendingEOSE, subid)
  }

  if sendClosed {
    s.WriteJSON(&[]interface{}{"CLOSED", subid, ""})
  }
}

func (s *Session) EVENT(data *[]interface{}) bool {
  event := (*data)[1].(map[string]interface{})
  id, ok := event["id"]
  if ok {
    s.WriteJSON(&[]interface{}{"OK", id, true, ""})
  }

  return ok
}
