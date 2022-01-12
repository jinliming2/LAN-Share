package main

import (
	"container/list"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jinliming2/LAN-Share/versions"
)

var (
	maxChatHistory   = flag.Int("history", 999, "Chat history count, mind the memory usage")
	messageSizeLimit = flag.Int("limit", 16*1024*1024, "The byte size limit per message, default to 16Mib, large file please send via 'file' option")
	address          = flag.String("addr", "[::]", "Listen on address")
	port             = flag.Int("port", 8080, "Listen on port")
	version          = flag.Bool("version", false, "Show version and exit")

	history = list.New()
)

func main() {
	flag.Parse()

	versions.PrintVersion()
	if *version {
		return
	}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", *address, *port),
		Handler:      HTTPHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	serverError := make(chan error, 1)
	go func() {
		log.Println("Server listing", server.Addr)
		serverError <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	select {
	case err := <-serverError:
		log.Println(err)
	case <-signals:
		log.Println("Gracefully exiting...")
		log.Println("Press Ctrl+C again to force exit.")
	}

	go func() {
		<-signals
		os.Exit(1)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
