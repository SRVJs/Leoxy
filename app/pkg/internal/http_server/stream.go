package http_server

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

const maxTCPConnections = 1024
const maxUDPSessions = 10000
const udpSessionIdleTimeout = 2 * time.Minute
const tcpIdleTimeout = 5 * time.Minute

type udpSession struct {
	connection   *net.UDPConn
	client       *net.UDPAddr
	lastActivity time.Time
}

func RunTCPProxy(listener net.Listener, target string, done <-chan struct{}) error {
	defer listener.Close()
	connections := make(chan struct{}, maxTCPConnections)
	go func() {
		<-done
		_ = listener.Close()
	}()

	for {
		client, err := listener.Accept()
		if err != nil {
			if isClosedError(err) {
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
	defer client.Close()
	upstream, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("TCP upstream %q connection failed: %v", target, err)
		return
	}
	defer upstream.Close()

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

func RunUDPProxy(conn *net.UDPConn, target string, done <-chan struct{}) error {
	defer conn.Close()
	upstreamAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return err
	}

	sessions := make(map[string]*udpSession)
	var sessionsMu sync.Mutex
	var sessionCount int
	go func() {
		<-done
		_ = conn.Close()
		sessionsMu.Lock()
		defer sessionsMu.Unlock()
		for _, session := range sessions {
			_ = session.connection.Close()
		}
	}()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				sessionsMu.Lock()
				for key, session := range sessions {
					if now.Sub(session.lastActivity) >= udpSessionIdleTimeout {
						delete(sessions, key)
						sessionCount--
						_ = session.connection.Close()
					}
				}
				sessionsMu.Unlock()
			}
		}
	}()

	buffer := make([]byte, 65535)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if isClosedError(err) {
				return nil
			}
			return err
		}

		key := clientAddr.String()
		sessionsMu.Lock()
		session := sessions[key]
		if session != nil {
			session.lastActivity = time.Now()
			sessionsMu.Unlock()
		} else if sessionCount >= maxUDPSessions {
			sessionsMu.Unlock()
			continue
		} else {
			sessionsMu.Unlock()
			upstream, dialErr := net.DialUDP("udp", nil, upstreamAddr)
			if dialErr != nil {
				log.Printf("UDP upstream %q connection failed: %v", target, dialErr)
				continue
			}
			sessionsMu.Lock()
			session = sessions[key]
			if session == nil && sessionCount < maxUDPSessions {
				session = &udpSession{connection: upstream, client: clientAddr, lastActivity: time.Now()}
				sessions[key] = session
				sessionCount++
				go relayUDPResponse(conn, key, session, &sessionsMu, sessions, &sessionCount)
			} else {
				_ = upstream.Close()
			}
			sessionsMu.Unlock()
			if session == nil {
				continue
			}
		}
		if _, err := session.connection.Write(buffer[:n]); err != nil {
			log.Printf("UDP upstream %q write failed: %v", target, err)
		}
	}
}

func relayUDPResponse(listener *net.UDPConn, key string, session *udpSession, sessionsMu *sync.Mutex, sessions map[string]*udpSession, sessionCount *int) {
	defer func() {
		sessionsMu.Lock()
		if sessions[key] == session {
			delete(sessions, key)
			*sessionCount = *sessionCount - 1
		}
		sessionsMu.Unlock()
	}()
	defer session.connection.Close()
	buffer := make([]byte, 65535)
	for {
		n, err := session.connection.Read(buffer)
		if err != nil {
			return
		}
		sessionsMu.Lock()
		session.lastActivity = time.Now()
		sessionsMu.Unlock()
		if _, err := listener.WriteToUDP(buffer[:n], session.client); err != nil {
			return
		}
	}
}

func isClosedError(err error) bool {
	return errors.Is(err, net.ErrClosed)
}
