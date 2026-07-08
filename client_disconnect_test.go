package ibapi

import (
	"net"
	"testing"
	"time"
)

// listenLoopback opens a listener that accepts and drops each connection.
// Returns the address the client should Dial and a stop function.
func listenLoopback(t *testing.T) (string, func()) {
	t.Helper()
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			// Hold the socket open until the client closes it; drain any
			// bytes so HandShake writes don't block on kernel buffers.
			go func(c net.Conn) {
				buf := make([]byte, 4096)
				for {
					if _, err := c.Read(buf); err != nil {
						c.Close()
						return
					}
				}
			}(c)
		}
	}()
	return l.Addr().String(), func() {
		l.Close()
		<-done
	}
}

// splitHostPort is a tiny helper to feed IbClient.Connect(host, port, id).
func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host:port: %v", err)
	}
	var port int
	if _, err := fmtSscan(portStr, &port); err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return host, port
}

// fmtSscan avoids importing fmt just for Sscan.
func fmtSscan(s string, out *int) (int, error) {
	var v int
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseErr{s}
		}
		v = v*10 + int(c-'0')
		n++
	}
	*out = v
	return n, nil
}

type parseErr struct{ s string }

func (e *parseErr) Error() string { return "not a number: " + e.s }

// TestDisconnectNoReceiver verifies Disconnect returns cleanly when no
// goroutine is parked on LoopUntilDone. Under the old TOCTOU guard this
// path was fine (the buggy send never fired), but under a naive fix that
// sends on an unbuffered channel it would deadlock.
func TestDisconnectNoReceiver(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	ic := NewIbClient(new(Wrapper))
	if err := ic.Connect(host, port, 1); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- ic.Disconnect() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Disconnect returned err: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Disconnect did not return within 2s (hang regression)")
	}
}

// TestDisconnectWithReceiver verifies a goroutine parked on ic.done
// unblocks when Disconnect runs. Under the buggy `len(ic.done) > 0` guard
// the deferred send never fires, so LoopUntilDone hangs forever even
// though Disconnect itself returns.
func TestDisconnectWithReceiver(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	ic := NewIbClient(new(Wrapper))
	if err := ic.Connect(host, port, 2); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	received := make(chan struct{})
	go func() {
		<-ic.done
		close(received)
	}()
	// Give the receiver a beat to park on the channel.
	time.Sleep(20 * time.Millisecond)

	if err := ic.Disconnect(); err != nil {
		t.Fatalf("Disconnect returned err: %v", err)
	}

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("receiver on ic.done did not unblock within 2s")
	}
}
