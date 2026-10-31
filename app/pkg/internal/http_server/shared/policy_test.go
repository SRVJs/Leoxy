package shared

import (
	"net"
	"testing"
)

func TestStreamPolicyDenyAllowAndLimit(t *testing.T) {
	policy, err := NewStreamPolicy([]string{"10.0.0.0/8"}, []string{"10.9.0.0/16"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	allowed := &net.TCPAddr{IP: net.ParseIP("10.1.1.1"), Port: 1000}
	denied := &net.TCPAddr{IP: net.ParseIP("10.9.1.1"), Port: 1000}
	outside := &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1000}

	if _, ok := policy.Acquire(denied); ok {
		t.Fatal("denied address accepted")
	}
	if _, ok := policy.Acquire(outside); ok {
		t.Fatal("address outside allow list accepted")
	}
	host, ok := policy.Acquire(allowed)
	if !ok {
		t.Fatal("allowed address rejected")
	}
	if _, ok := policy.Acquire(allowed); ok {
		t.Fatal("per-ip limit not enforced")
	}
	policy.Release(host)
	if _, ok := policy.Acquire(allowed); !ok {
		t.Fatal("slot not released")
	}
}

func TestNilStreamPolicyAllowsEverything(t *testing.T) {
	var policy *StreamPolicy
	if _, ok := policy.Acquire(&net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1}); !ok {
		t.Fatal("nil policy must allow")
	}
	policy.Release("192.0.2.1")
}
