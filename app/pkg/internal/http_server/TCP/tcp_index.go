package tcp

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/http_server/shared"
)

const maxTCPConnections = 1024
const tcpIdleTimeout = 5 * time.Minute

func RunTCPProxy(listener net.Listener, target string, done <-chan struct{}) error {
	defer func() {
		if err := listener.Close(); err != nil {
			log.Printf("close TCP listener: %v", err)
		}
	}()
	connections := make(chan struct{}, maxTCPConnections)
	go func() {
		<-done
		_ = listener.Close()
	}()

	for {
		client, err := listener.Accept()
		if err != nil {
			if shared.IsClosedError(err) {
				return nil
			}
			return err
		}
		select {
		case connections <- struct{}{}:
			go func() {
				defer func() { <-connections }()
				proxyTCPConnection(client, target)
			}()
		default:
			_ = client.Close()
		}
	}
}

func proxyTCPConnection(client net.Conn, target string) {
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close TCP client connection: %v", err)
		}
	}()
	upstream, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("TCP upstream %q connection failed: %v", target, err)
		return
	}
	defer func() {
		if err := upstream.Close(); err != nil {
			log.Printf("close TCP upstream connection: %v", err)
		}
	}()

	_ = client.SetDeadline(time.Now().Add(tcpIdleTimeout))
	_ = upstream.SetDeadline(time.Now().Add(tcpIdleTimeout))
	var copies sync.WaitGroup
	copies.Add(2)
	go func() {
		defer copies.Done()
		copyTCP(upstream, client, client, upstream)
		if conn, ok := upstream.(*net.TCPConn); ok {
			_ = conn.CloseWrite()
		}
	}()
	go func() {
		defer copies.Done()
		copyTCP(client, upstream, client, upstream)
		if conn, ok := client.(*net.TCPConn); ok {
			_ = conn.CloseWrite()
		}
	}()
	copies.Wait()
}

func copyTCP(destination, source, client, upstream net.Conn) {
	buffer := make([]byte, 32*1024)
	for {
		count, readErr := source.Read(buffer)
		if count > 0 {
			deadline := time.Now().Add(tcpIdleTimeout)
			_ = client.SetDeadline(deadline)
			_ = upstream.SetDeadline(deadline)
			written := 0
			for written < count {
				writeCount, err := destination.Write(buffer[written:count])
				written += writeCount
				if err != nil || writeCount == 0 {
					return
				}
			}
		}
		if readErr != nil {
			return
		}
	}
}
