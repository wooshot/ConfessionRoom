package main

import "github.com/wooshot/ConfessionRoom/server"

func main() {
	c := server.NewConfessionServer()
	g := server.NewConfessionServer()
	c.Start()
	g.Start()
}
