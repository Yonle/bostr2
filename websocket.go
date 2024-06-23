package main

import (
  "net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
  ReadBufferSize:    1024,
  WriteBufferSize:   1024,
  CheckOrigin: func(_ *http.Request) bool { return true },
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
    Relays: make(SessionRelays),
    UpstreamMessage: make(SessionUpstreamMessage),
  }

  conn.SetCloseHandler(sess.Destroy)

  go func() {
    for msg := range sess.UpstreamMessage {
      if err := conn.WriteMessage(websocket.TextMessage, *msg); err != nil {
        return
      }
    }
  }()

  defer conn.Close()

  for {
    var json []interface{}
    if err := conn.ReadJSON(&json); err != nil {
      sess.Destroy(0, "")
      close(sess.UpstreamMessage)
      return
    }

    switch json[0].(string) {
    case "REQ":
      sess.REQ(&json)
    case "CLOSE":
      sess.CLOSE(&json, true)
    case "EVENT":
      sess.EVENT(&json)
    }

  }
}
