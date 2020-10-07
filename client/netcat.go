package main

import (
	"fmt"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	userName := flag.String("name", "user", "User's name.")
	roomId := flag.String("roomID", "00000000-0000-0000-0000-000000000000", "Chat room id (uuid)")

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

	enter := strings.NewReader(fmt.Sprintf("%s:%s\n", *userName, *roomId))
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
