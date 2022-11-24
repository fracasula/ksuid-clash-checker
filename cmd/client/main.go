package main

import (
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
		log.Println("please specify server address and number of threads")
		log.Println("example:")
		log.Println("go run main.go 192.168.1.31:8181 20")
		os.Exit(1)
	}

	noOfThreads, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		log.Fatalf("invalid number of threads specified: %q", os.Args[2])
	}

	serverAddress := os.Args[1]
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("failed to dial %q: %v", serverAddress, err)
	}
	defer func() { _ = conn.Close() }()

	log.Printf("connected to %s", os.Args[1])

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	wg := sync.WaitGroup{}
	ch := make(chan string, 10000000)

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

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case id := <-ch:
			_, err := conn.Write([]byte(id + "\n"))
			if err != nil {
				log.Println("failed to write id:", err)
				cancel()
				break loop
			}
		}
	}

	wg.Wait()
}
