package main

import (
	"log"
	"net/http"
	"sync"
)

var ips = make(map[string]int)
var mu sync.Mutex

func ConnPerIPRateLimit_Pass(ip string, w http.ResponseWriter) (pass bool) {
	if config.MaxConnectionsPerIP < 1 {
		return true
	}

	mu.Lock()
	defer mu.Unlock()

	conns, ok := ips[ip]
	if !ok {
		conns = 0
	}

	if conns >= config.MaxConnectionsPerIP {
		TooManyRequest(w)
		log.Printf("%s rejected: Too many open connections", ip)
		return false
	}

	conns++
	ips[ip] = conns
	return true
}

func ConnPerIPRateLimit_OnDisconnect(ip string) {
	if config.MaxConnectionsPerIP < 1 {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	conns, ok := ips[ip]
	if !ok {
		return
	}

	conns--

	if conns < 1 {
		delete(ips, ip)
	} else {
		ips[ip] = conns
	}
}
