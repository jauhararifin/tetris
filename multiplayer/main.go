package main

import "flag"

func main() {
	isServer := flag.Bool("server", false, "run server")
	host := flag.String("host", "localhost:8123", "host")
	name := flag.String("name", "", "name")
	room := flag.String("room", "", "room")
	flag.Parse()

	if *isServer {
		startServer()
	} else {
		startClient(*host, *name, *room)
	}
}

