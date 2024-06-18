package main

import (
  "fmt"
  "net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
}

func Accept_Websocket (w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

  if err != nil {
    return
  }

  var sess = Session{
    Owner: conn,
    Sub_IDs: make(SessionSubIDs),
    Event_IDs: make(SessionEventIDs),
    PendingEOSE: make(SessionPendingEOSE),
    Relays: []*websocket.Conn{},
  }

  bouncer[conn] = &sess

  sess.StartConnect()

  go func() {
    defer conn.Close()
    defer delete(bouncer, conn)

    for {
      var json []interface{}
      if err := conn.ReadJSON(&json); err != nil {
        conn.WriteJSON([2]string{"NOTICE", fmt.Sprintf("ошибка: %s. отключение", err)})
        break
      }

      switch json[0].(string) {
      case "REQ":
        sess.REQ(&json)
        sess.Broadcast(&json)
      case "CLOSE":
        sess.CLOSE(&json, true)
        sess.Broadcast(&json)
      case "EVENT":
        if invalid := sess.EVENT(&json); !invalid {
          conn.WriteJSON([2]string{"NOTICE", "Неверный объект."})
        } else {
          sess.Broadcast(&json)
        }
      }
    }
  }()
}
