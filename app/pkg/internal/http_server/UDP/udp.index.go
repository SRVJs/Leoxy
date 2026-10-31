package udp

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/http_server/shared"
)

const udpSessionIdleTimeout = 5 * time.Minute
const maxUDPSessions = 4096

type udpSession struct {
	connection   *net.UDPConn
	client       *net.UDPAddr
	host         string
	lastActivity time.Time
}

type sessionTable struct {
	mutex    sync.Mutex
	sessions map[string]*udpSession
	policy   *shared.StreamPolicy
}

func RunUDPProxy(conn *net.UDPConn, target string, policy *shared.StreamPolicy, done <-chan struct{}) error {
	defer func() {
		if err := conn.Close(); err != nil && !shared.IsClosedError(err) {
			log.Printf("UDP listener close failed: %v", err)
		}
	}()
	upstreamAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return err
	}

	table := &sessionTable{sessions: make(map[string]*udpSession), policy: policy}
	go func() {
		<-done
		_ = conn.Close()
		table.closeAll()
	}()
	go table.sweep(done)

	buffer := make([]byte, 65535)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if shared.IsClosedError(err) {
				return nil
			}
			return err
		}

		session := table.lookup(clientAddr.String())
		if session == nil {
			session = table.create(conn, clientAddr, upstreamAddr, target)
			if session == nil {
				continue
			}
		}
		if _, err := session.connection.Write(buffer[:n]); err != nil && !shared.IsClosedError(err) {
			log.Printf("UDP upstream %q write failed: %v", target, err)
		}
	}
}

func (t *sessionTable) lookup(key string) *udpSession {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	session := t.sessions[key]
	if session != nil {
		session.lastActivity = time.Now()
	}
	return session
}

func (t *sessionTable) touch(session *udpSession) {
	t.mutex.Lock()
	session.lastActivity = time.Now()
	t.mutex.Unlock()
}

func (t *sessionTable) create(listener *net.UDPConn, client, upstream *net.UDPAddr, target string) *udpSession {
	t.mutex.Lock()
	full := len(t.sessions) >= maxUDPSessions
	t.mutex.Unlock()
	if full {
		return nil
	}
	host, ok := t.policy.Acquire(client)
	if !ok {
		return nil
	}
	connection, err := net.DialUDP("udp", nil, upstream)
	if err != nil {
		t.policy.Release(host)
		log.Printf("UDP upstream %q connection failed: %v", target, err)
		return nil
	}
	session := &udpSession{connection: connection, client: client, host: host, lastActivity: time.Now()}
	key := client.String()

	t.mutex.Lock()
	if len(t.sessions) >= maxUDPSessions {
		t.mutex.Unlock()
		_ = connection.Close()
		t.policy.Release(host)
		return nil
	}
	t.sessions[key] = session
	t.mutex.Unlock()

	go t.relay(listener, key, session)
	return session
}

func (t *sessionTable) remove(key string, session *udpSession) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.sessions[key] != session {
		return
	}
	delete(t.sessions, key)
	t.policy.Release(session.host)
}

func (t *sessionTable) closeAll() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for _, session := range t.sessions {
		_ = session.connection.Close()
	}
}

func (t *sessionTable) sweep(done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case now := <-ticker.C:
			t.mutex.Lock()
			for key, session := range t.sessions {
				if now.Sub(session.lastActivity) >= udpSessionIdleTimeout {
					delete(t.sessions, key)
					t.policy.Release(session.host)
					_ = session.connection.Close()
				}
			}
			t.mutex.Unlock()
		}
	}
}

func (t *sessionTable) relay(listener *net.UDPConn, key string, session *udpSession) {
	defer t.remove(key, session)
	defer func() {
		if err := session.connection.Close(); err != nil && !shared.IsClosedError(err) {
			log.Printf("UDP upstream close failed: %v", err)
		}
	}()
	buffer := make([]byte, 65535)
	for {
		n, err := session.connection.Read(buffer)
		if err != nil {
			return
		}
		t.touch(session)
		if _, err := listener.WriteToUDP(buffer[:n], session.client); err != nil {
			return
		}
	}
}
