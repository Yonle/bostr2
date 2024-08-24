package main

import (
	"fmt"
	"net/http"
)

/*
func HijackAndKill(w http.ResponseWriter) {
	hj := w.(http.Hijacker)
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}

	conn.Close()
}
*/

func TooManyRequest(w http.ResponseWriter) {
	h := w.Header()
	h.Add("Content-Type", "text/plain")
	w.WriteHeader(429)
	fmt.Fprint(w, "Too many open connections from this IP. Close an existing open socket and try again.")
}
