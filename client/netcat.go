package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	args := os.Args
	var name string
	if len(args) == 1 {
		fmt.Println("Please give the name")
		os.Exit(1)
	}
	name = args[1]

	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, conn) // NOTE: ignoring errors
		log.Println("done")
		done <- struct{}{} // signal the main goroutine
	}()

	enter := strings.NewReader(fmt.Sprintf("%s\n", name))
	if _, err := io.Copy(conn, enter); err != nil {
		log.Fatal(err)
	}
	mustCopy(conn, os.Stdin)
	conn.Close()
	<-done // wait for background goroutine to finish
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
