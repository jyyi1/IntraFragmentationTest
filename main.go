package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	listenAddr := flag.String("addr", ":60443", "Address and port to listen on")
	readTimeout := flag.Duration("timeout", 15*time.Second, "Timeout for reading initial data from client")
	flag.Parse()

	chlogger := TLSHelloRecordLogger{}

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		slog.Error("Failed to start TCP listener", slog.String("address", *listenAddr), slog.Any("error", err))
		os.Exit(1)
	}
	defer listener.Close()
	slog.Info("TCP listener started", slog.String("address", listener.Addr().String()))

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// WaitGroup to track active connections
	var wg sync.WaitGroup

	// Start Accept Loop Goroutine
	go func() {
		slog.Info("Accept loop started. Waiting for connections...")
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					slog.Info("Listener closed, accept loop terminating.")
					return
				} else {
					slog.Error("Error accepting connection", slog.Any("error", err))
					continue
				}
			}
			go handleTCPConnection(conn, chlogger, *readTimeout, &wg)
		}
	}()

	// Wait for Shutdown Signal
	sig := <-signalChan
	slog.Info("Received signal, initiating graceful shutdown...", slog.String("signal", sig.String()))

	slog.Info("Closing listener to stop accepting new connections...")
	if err := listener.Close(); err != nil {
		slog.Error("Error closing listener", slog.Any("error", err))
	}

	slog.Info("Waiting for active connections to finish...")
	wg.Wait()

	slog.Info("All connections finished. Shutdown complete.")
}

type TLSHelloRecordLogger struct {
}

func (TLSHelloRecordLogger) WriteRecord(remoteAddr string, data []byte) {
	slog.Info("Received TLS ClientHello record",
		slog.String("remote_addr", remoteAddr),
		slog.String("data_hex", hex.EncodeToString(data)),
		slog.Int("raw_data_len", len(data)),
	)
}
