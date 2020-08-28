package main

import "github.com/wooshot/grpcTest/server"

func main() {
	g := server.New()
	g.Start()
}
