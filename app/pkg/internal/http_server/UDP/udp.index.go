package udp

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/http_server/shared"
)

const udpSessionIdleTimeout = 5 * time.Minute
const maxUDPSessions = 10000

type udpSession struct {
	connection   *net.UDPConn
	client       *net.UDPAddr
	lastActivity time.Time
}

func RunUDPProxy(conn *net.UDPConn, target string, done <-chan struct{}) error {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("close UDP listener: %v", err)
		}
	}()
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
			if shared.IsClosedError(err) {
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
	defer func() {
		if err := session.connection.Close(); err != nil {
			log.Printf("close UDP session connection: %v", err)
		}
	}()
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
