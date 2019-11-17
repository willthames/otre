package main

import (
	"flag"
	"net/http"
	"strconv"
)

type app struct {
	Port   string
	server *http.Server
}

func cliParse() *app {
	port := flag.Int("port", 8080, "server port")
	flag.Parse()
	a := &app{
		Port: strconv.Itoa(*port),
	}
	return a
}
