package main

import "github.com/wooshot/ConfessionRoom/server"

func main() {
	g := server.New()
	g.Start()
}
