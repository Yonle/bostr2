package main

import (
	"log"
	"net/http"
)

var ips = make(map[string]int)

func ConnPerIPRateLimit_Pass(ip string, w http.ResponseWriter) (pass bool) {
	if config.MaxConnectionsPerIP < 1 {
		return true
	}

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
