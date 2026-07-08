package ibapi

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingWrapper records ConnectionClosed calls so tests can catch
// double-fires on the Disconnect idempotency path.
type countingWrapper struct {
	Wrapper
	closed atomic.Int32
}

func (w *countingWrapper) ConnectionClosed() { w.closed.Add(1) }
func (w *countingWrapper) closedCount() int  { return int(w.closed.Load()) }

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

// TestConcurrentDisconnect verifies two Disconnect calls firing in
// parallel don't race on reset() or double-fire ConnectionClosed. This
// is the exact scenario when goReceive's post-loop branch spawns
// `go ic.Disconnect()` while the caller also explicitly disconnects.
// Only meaningful under -race.
func TestConcurrentDisconnect(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	// countingWrapper flags a second ConnectionClosed as a regression.
	cw := &countingWrapper{}
	ic := NewIbClient(cw)
	if err := ic.Connect(host, port, 3); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	const N = 8
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() { defer wg.Done(); ic.Disconnect() }()
	}
	waitCh := make(chan struct{})
	go func() { wg.Wait(); close(waitCh) }()
	select {
	case <-waitCh:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent Disconnect did not settle within 3s")
	}
	if got := cw.closedCount(); got != 1 {
		t.Fatalf("ConnectionClosed fired %d times; want 1", got)
	}
}

// TestFastRunThenDisconnect stresses the wg.Add-before-go fix. Under
// the old pattern where each worker called wg.Add(1) inside itself,
// a fast Disconnect after Run could hit wg.Wait() with counter 0
// (workers not yet scheduled), let Wait return, and race the workers
// against reset(). Under -race the schedule-then-Wait race would
// eventually catch a field write. Loop enough times to make the
// scheduler-dependent race likely to hit.
func TestFastRunThenDisconnect(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	for i := 0; i < 50; i++ {
		ic := NewIbClient(new(Wrapper))
		if err := ic.Connect(host, port, int64(100+i)); err != nil {
			t.Fatalf("iter %d Connect: %v", i, err)
		}
		// Fake the "connected" state Run requires without a real
		// HandShake; Run only checks IsConnected() then spawns.
		ic.setConnState(CONNECTED)
		if err := ic.Run(); err != nil {
			t.Fatalf("iter %d Run: %v", i, err)
		}
		if err := ic.Disconnect(); err != nil {
			t.Fatalf("iter %d Disconnect: %v", i, err)
		}
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
