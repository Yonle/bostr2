package main

import (
	"slices"
	"strconv"
	"time"

	"github.com/coder/websocket/wsjson"
	"github.com/nbd-wtf/go-nostr"
)

func (s *Session) doAuth() {
	if s.authID == "" {
		s.authID = strconv.FormatInt(time.Now().UnixMicro(), 10)
	}

	wsjson.Write(s.ctx, s.conn, []string{"AUTH", s.authID})
}

func (s *Session) verifyAuth(ev nostr.Event) (bool, string) {
	if time.Since(ev.CreatedAt.Time()) > time.Minute {
		return false, "this event is 1 minute late."
	}

	if !ev.CheckID() {
		return false, "invalid event"
	}

	if ok, _ := ev.CheckSignature(); !ok {
		return false, "invalid signature"
	}

	if ev.Kind != 22242 {
		return false, "not 22242 kind"
	}

	if !slices.Contains(config.AllowedPubkeys, ev.PubKey) {
		return false, "unauthorized"
	}

	if !ev.Tags.ContainsAny("challenge", []string{s.authID}) {
		return false, "invalid challenge value"
	}

	if !ev.Tags.ContainsAny("relay", []string{config.MyAddress}) {
		return false, "the relay field does not match to the current connection: " + ev.Tags.Find("relay").Value()
	}

	s.authed = true
	return true, ""
}
