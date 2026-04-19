package main

import (
	"net/http"

	"github.com/tpyle/websnake"
)

func main() {
	ws := websnake.NewWebSnake(websnake.AllowMissingKeys(true))

	muxer := ws.Handler()

	http.ListenAndServe(":8080", muxer)
}
