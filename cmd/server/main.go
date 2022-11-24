package main

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"

	"github.com/segmentio/ksuid"
)

func main() {
	if len(os.Args) < 3 {
		log.Println("please specify tcp port to listen to and number of threads")
		log.Println("example:")
		log.Println("go run main.go 8181 20")
		os.Exit(1)
	}

	noOfThreads, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		log.Fatalf("invalid number of threads specified: %q", os.Args[2])
	}

	tcpPort, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("invalid tcp port specified: %q", os.Args[1])
	}

	l, err := net.Listen("tcp", ":"+strconv.FormatInt(tcpPort, 10))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", tcpPort, err)
	}
	defer func() { _ = l.Close() }()

	log.Printf("listening on %d", tcpPort)

	wg := sync.WaitGroup{}
	ch := make(chan string, 10000000)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; ; i++ {
			if ctx.Err() != nil {
				return
			}
			conn, err := l.Accept()
			if err != nil {
				log.Println("failed to accept connection:", err)
				return
			}

			log.Println("connection accepted")

			wg.Add(1)
			go func(i int, conn net.Conn) {
				defer wg.Done()
				defer func() { _ = conn.Close() }()

				reader := bufio.NewReader(conn)

				for {
					select {
					case <-ctx.Done():
						return
					default:
						line, _, err := reader.ReadLine()
						if err != nil {
							log.Printf("cannot read from connection: %v", err)
							cancel()
							_ = conn.Close()
							return
						}
						if bytes.Contains(line, []byte("\n")) {
							log.Printf("error: data from conn %d contains new line", i)
							cancel()
							_ = conn.Close()
							return
						}
						ch <- string(line)
					}
				}
			}(i, conn)
		}
	}()

	wg.Add(int(noOfThreads))
	for i := 0; i < int(noOfThreads); i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case ch <- ksuid.New().String():
				}
			}
		}()
	}

	m := make(map[string]struct{}, 10000000)
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case id := <-ch:
			if _, ok := m[id]; ok {
				log.Println("duplicate id:", id)
				continue
			}
			m[id] = struct{}{}
		}
	}

	wg.Wait()
}
