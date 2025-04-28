package main

import (
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

func handleTCPConnection(conn net.Conn, logger TLSHelloRecordLogger, readTimeout time.Duration, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	slog.Info("Accepted connection", slog.String("remote_addr", remoteAddr))

	err := conn.SetReadDeadline(time.Now().Add(readTimeout))
	if err != nil {
		slog.Error("Error setting read deadline", slog.String("remote_addr", remoteAddr), slog.Any("error", err))
		return
	}

	buffer := make([]byte, 4096)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			slog.Error("Critical: Error setting read deadline. Closing connection.",
				slog.String("remote_addr", remoteAddr),
				slog.Any("error", err))
			return
		}

		n, err := conn.Read(buffer)
		logger.WriteRecord(remoteAddr, buffer[:n])

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				slog.Info("Read timeout: No further data from client. Closing connection.",
					slog.String("remote_addr", remoteAddr))
			} else if err == io.EOF {
				slog.Info("Connection closed by client (EOF).",
					slog.String("remote_addr", remoteAddr))
			} else {
				slog.Error("Error reading from client. Closing connection.",
					slog.String("remote_addr", remoteAddr),
					slog.Any("error", err))
			}
			break
		}
	}

	slog.Info("Closing connection.", slog.String("remote_addr", remoteAddr))
}
