package main

import (
	"context"
	"log"
	"sync"
	"time"

/*	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"*/

	"github.com/Yonle/bostr2/relayHandler"
)

type SessionEvents map[string]map[string]struct{}
type SessionEOSEs map[string]int
type SessionSubs map[string]*[]relayHandler.Message

type Session struct {
	ClientIP string

	// these three Client* channels are for receiving messages from client
	ClientREQ     relayHandler.MessageChan
	ClientCLOSE   relayHandler.MessageChan
	ClientEVENT   relayHandler.MessageChan
	clientMessage relayHandler.MessageChan // to sent to upstreams

	UpMessage relayHandler.MessageChan // to sent to client

	events        SessionEvents
	pendingEOSE   SessionEOSEs
	subscriptions SessionSubs

	relay      relayHandler.RelaySession
	listenerWg sync.WaitGroup

	destroyed chan struct{}
	ctx       context.Context
}

func (s *Session) Start() {
	// connect first
	log.Println("Vrooom..")
	s.relay.Init(config.Relays)
	log.Println("Dodonk")

	//s.listenerWg.Add(2)

	// deal with destroy request.
	go func() {
	listener:
		for {
			select {
			case <-s.ctx.Done():
				log.Println("nuin")
				go func() {
					time.Sleep(5 * time.Second)
					select {
					case <-s.destroyed:
					default:
						panic("something hangs.")
					}
				}()
				s.relay.Wait()

				close(s.destroyed)

				//s.listenerWg.Wait()

				close(s.ClientREQ)
				close(s.ClientCLOSE)
				close(s.ClientEVENT)
				close(s.clientMessage)

				close(s.UpMessage)

				log.Println(s.ClientIP, "=================== clean shutdown finished.")
				break listener
			}
		}
	}()
}
