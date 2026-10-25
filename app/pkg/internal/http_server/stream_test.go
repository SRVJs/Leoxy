package http_server

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestRunTCPProxyForwardsData(t *testing.T) {
	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer upstreamListener.Close()
	go func() {
		connection, acceptErr := upstreamListener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()
		_, _ = io.Copy(connection, connection)
	}()

	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	proxyStopped := make(chan error, 1)
	go func() {
		proxyStopped <- RunTCPProxy(proxyListener, upstreamListener.Addr().String(), done)
	}()

	client, err := net.DialTimeout("tcp", proxyListener.Addr().String(), time.Second)
	if err != nil {
		close(done)
		t.Fatal(err)
	}
	_ = client.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := client.Write([]byte("ping")); err != nil {
		client.Close()
		close(done)
		t.Fatal(err)
	}
	response := make([]byte, 4)
	if _, err := io.ReadFull(client, response); err != nil {
		client.Close()
		close(done)
		t.Fatal(err)
	}
	client.Close()
	close(done)
	select {
	case err := <-proxyStopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TCP proxy did not stop")
	}
	if string(response) != "ping" {
		t.Fatalf("response = %q, want ping", response)
	}
}

func TestRunUDPProxyForwardsData(t *testing.T) {
	upstream, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()
	go func() {
		buffer := make([]byte, 64)
		count, client, readErr := upstream.ReadFromUDP(buffer)
		if readErr == nil {
			_, _ = upstream.WriteToUDP(buffer[:count], client)
		}
	}()

	proxy, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	proxyStopped := make(chan error, 1)
	go func() {
		proxyStopped <- RunUDPProxy(proxy, upstream.LocalAddr().String(), done)
	}()

	client, err := net.DialUDP("udp", nil, proxy.LocalAddr().(*net.UDPAddr))
	if err != nil {
		close(done)
		t.Fatal(err)
	}
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := client.Write([]byte("ping")); err != nil {
		close(done)
		t.Fatal(err)
	}
	response := make([]byte, 64)
	count, err := client.Read(response)
	if err != nil {
		close(done)
		t.Fatal(err)
	}
	close(done)
	select {
	case err := <-proxyStopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("UDP proxy did not stop")
	}
	if string(response[:count]) != "ping" {
		t.Fatalf("response = %q, want ping", response[:count])
	}
}
