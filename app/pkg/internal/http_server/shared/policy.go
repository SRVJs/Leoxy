package shared

import (
	"net"
	"sync"

	"github.com/elrefai99/Leoxy/app/pkg/internal/common"
)

type StreamPolicy struct {
	allow    []*net.IPNet
	deny     []*net.IPNet
	maxPerIP int
	mutex    sync.Mutex
	active   map[string]int
}

func NewStreamPolicy(allow, deny []string, maxPerIP int) (*StreamPolicy, error) {
	allowed, err := common.ParseCIDRs(allow)
	if err != nil {
		return nil, err
	}
	denied, err := common.ParseCIDRs(deny)
	if err != nil {
		return nil, err
	}
	return &StreamPolicy{allow: allowed, deny: denied, maxPerIP: maxPerIP, active: make(map[string]int)}, nil
}

func (p *StreamPolicy) Acquire(addr net.Addr) (string, bool) {
	host := addrHost(addr)
	if p == nil {
		return host, true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", false
	}
	for _, network := range p.deny {
		if network.Contains(ip) {
			return "", false
		}
	}
	if len(p.allow) > 0 && !within(p.allow, ip) {
		return "", false
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.maxPerIP > 0 && p.active[host] >= p.maxPerIP {
		return "", false
	}
	p.active[host]++
	return host, true
}

func (p *StreamPolicy) Release(host string) {
	if p == nil {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.active[host] <= 1 {
		delete(p.active, host)
		return
	}
	p.active[host]--
}

func addrHost(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}

func within(networks []*net.IPNet, ip net.IP) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
