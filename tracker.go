// tracker.go
// NOC-02 Tracker System : Broadcasting

package main

import (
	"context"
	"encoding/json"
	"time"

	"codeberg.org/Yonle/bostr2/relayHandler"
)

var tracker_submit_cmd []json.RawMessage

func prepareTrackers() {
	if len(config.MyAddress) == 0 || len(config.Trackers) < 1 {
		return
	}

	trackers := relayHandler.NewSession(context.Background(), "")
	ticker := time.Tick(time.Second * 30)
	data := []json.RawMessage{[]byte("SUBMIT"), []byte(config.MyAddress)}

	trackers.Init(config.Trackers)

	go dealWithTrackers(trackers, ticker, data)
}

func dealWithTrackers(trackers *relayHandler.RelaySession, ticker <-chan time.Time, data []json.RawMessage) {
	for range ticker {
		trackers.Broadcast(tracker_submit_cmd)
	}
}
