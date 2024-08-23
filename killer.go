package main

import (
	"net/http"
)

func HijackAndKill(w http.ResponseWriter) {
	hj := w.(http.Hijacker)
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}

	conn.Close()
}
