package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

const (
	bufSize       = 4 * 1024           // 4 KiB
	progressSmall = 1024 * 1024        // 1 MiB
	progressLarge = 32 * progressSmall // 64 MiB
)

var (
	_seed     = flag.Int("seed", 123456, "The seed for the stream, must match on server and client")
	_isServer = flag.Bool("s", false, "Whether to run as a server")
	_clientHP = flag.String("c", "", "The host:port of the server to connect to")
)

func main() {
	flag.Parse()

	isServer := *_isServer
	isClient := *_clientHP != ""

	// Must only specify one of isServer or isClient
	if isServer == isClient {
		log.Fatal("Must specify one of -s or -c.")
	}

	switch {
	case isServer:
		runServer()
	case isClient:

		runClient(*_clientHP)
	}
}

func runServer() {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}
	fmt.Println("Listening on", ln.Addr())

	conn, err := ln.Accept()
	if err != nil {
		log.Fatal("Failed to accept connection", err)
	}

	fmt.Println("Received connection from", conn.RemoteAddr())
	handleConn(conn)
}

func getStream() io.Reader {
	return rand.New(rand.NewSource(int64(*_seed)))
}

func runClient(hostPort string) {
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		log.Fatal("Failed to connect to server:", err)
	}

	handleConn(conn)
}

func handleConn(conn net.Conn) {
	go io.Copy(conn, getStream())
	verifyRead(conn, getStream())
}

func verifyRead(conn net.Conn, s io.Reader) {
	var (
		connBuf   = make([]byte, bufSize)
		streamBuf = make([]byte, bufSize)

		p = newProgress()
	)

	for {
		n, err := conn.Read(connBuf)
		if err != nil {
			log.Fatalf("read from conn %v failed: %v", conn.RemoteAddr(), err)
		}

		_, err = io.ReadFull(s, streamBuf[:n])
		if err != nil {
			log.Fatal("Failed to read stream:", err)
		}

		if !bytes.Equal(connBuf[:n], streamBuf[:n]) {
			log.Fatal("Mismatch on read bytes!")
		}

		p.update(n)
		time.Sleep(time.Millisecond)
	}
}

type progress struct {
	cur       int
	nextSmall int
	nextLarge int
}

func newProgress() *progress {
	return &progress{nextSmall: progressSmall, nextLarge: progressLarge}
}

func (p *progress) update(n int) {
	p.cur += n
	if p.cur > p.nextSmall {
		fmt.Print(".")
		p.nextSmall += progressSmall
	}
	if p.cur > p.nextLarge {
		fmt.Println()
		p.nextLarge += progressLarge
	}
}
