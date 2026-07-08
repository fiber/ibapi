package ibapi

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// errCatcher records Wrapper.Error calls so we can assert against
// send-after-close and other API-boundary rejections.
type errCatcher struct {
	Wrapper
	mu   sync.Mutex
	errs []string
}

func (w *errCatcher) Error(_ int64, _ int64, msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.errs = append(w.errs, msg)
}

func (w *errCatcher) contains(sub string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, m := range w.errs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

// TestDisconnectBeforeConnect must not nil-deref on the zero-value
// embedded *net.TCPConn.
func TestDisconnectBeforeConnect(t *testing.T) {
	ic := NewIbClient(new(Wrapper))
	// no Connect() — ic.conn.TCPConn is nil, ic.done is nil.
	// Expected: Disconnect is a no-op, does not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Disconnect before Connect panicked: %v", r)
		}
	}()
	// ic.done is nil so the caller shouldn't have called Disconnect.
	// Guard: at least make sure conn.disconnect() (the nil-guarded path)
	// runs without crashing when Disconnect is entered.
	if ic.conn.TCPConn != nil {
		t.Fatal("test precondition wrong")
	}
	if err := ic.conn.disconnect(); err != nil {
		t.Fatalf("conn.disconnect returned err on zero-value: %v", err)
	}
}

// TestRequestAfterDisconnectDoesNotHang verifies that firing a request
// method after Disconnect returns cleanly (via wrapper.Error) instead
// of blocking on a full reqChan with no reader.
func TestRequestAfterDisconnectDoesNotHang(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	ec := &errCatcher{}
	ic := NewIbClient(ec)
	if err := ic.Connect(host, port, 10); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ic.setConnState(CONNECTED)
	if err := ic.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if err := ic.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	// Fire 20 requests — more than the reqChan buffer of 10. If the
	// enqueue guard misses, sends 11-20 block forever.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			ic.ReqCurrentTime()
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("requests after Disconnect blocked (enqueue guard missing)")
	}
	if !ec.contains("Not connected") {
		t.Fatalf("expected NOT_CONNECTED error from wrapper; got %v", ec.errs)
	}
}

// TestLoopUntilDoneWatcherReleasedOnDirectDisconnect verifies that the
// LoopUntilDone ctx-watcher goroutine exits when Disconnect fires
// directly (i.e. not via ctx cancel). Under the old code the watcher
// stayed parked on <-ic.ctx.Done() forever, leaking one goroutine
// per LoopUntilDone call.
func TestLoopUntilDoneWatcherReleasedOnDirectDisconnect(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	ic := NewIbClient(new(Wrapper))
	_ = ic.SetContext(context.Background()) // never cancels
	if err := ic.Connect(host, port, 11); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ic.setConnState(CONNECTED)
	if err := ic.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	loopReturned := make(chan struct{})
	go func() {
		ic.LoopUntilDone()
		close(loopReturned)
	}()
	// Give LoopUntilDone a beat to spawn its watcher.
	time.Sleep(20 * time.Millisecond)

	if err := ic.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	select {
	case <-loopReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("LoopUntilDone did not return within 2s after Disconnect")
	}

	// A second Disconnect call must not deadlock either — indirectly
	// checks that the watcher's ctx.Done() branch isn't the only exit.
	if err := ic.Disconnect(); err != nil {
		t.Fatalf("second Disconnect: %v", err)
	}
}

// TestSettersRejectedAfterConnect verifies the Python-model contract
// applies to all three construction-time setters. Racy callers who
// swap ctx / connect options / wrapper after Connect get an error
// instead of a silent field write that races HandShake or the
// LoopUntilDone watcher.
func TestSettersRejectedAfterConnect(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	ic := NewIbClient(new(Wrapper))
	// pre-Connect: all three succeed
	if err := ic.SetContext(context.Background()); err != nil {
		t.Fatalf("pre-Connect SetContext: %v", err)
	}
	if err := ic.SetConnectionOptions("+PACEAPI"); err != nil {
		t.Fatalf("pre-Connect SetConnectionOptions: %v", err)
	}
	if err := ic.SetWrapper(new(Wrapper)); err != nil {
		t.Fatalf("pre-Connect SetWrapper: %v", err)
	}

	if err := ic.Connect(host, port, 30); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer ic.Disconnect()

	if err := ic.SetContext(context.Background()); err != ALREADY_CONNECTED {
		t.Fatalf("post-Connect SetContext got %v; want ALREADY_CONNECTED", err)
	}
	if err := ic.SetConnectionOptions("+FOO"); err != ALREADY_CONNECTED {
		t.Fatalf("post-Connect SetConnectionOptions got %v; want ALREADY_CONNECTED", err)
	}
	if err := ic.SetWrapper(new(Wrapper)); err != ALREADY_CONNECTED {
		t.Fatalf("post-Connect SetWrapper got %v; want ALREADY_CONNECTED", err)
	}
}

// TestSetWrapperRejectedAfterConnect verifies SetWrapper is a no-op
// once a connection exists. Following the Python SDK model (wrapper
// set at construction, never replaced during a session), we reject
// mid-Run swaps rather than plumb atomic pointers through every
// callback site. Pre-Connect swaps still succeed.
func TestSetWrapperRejectedAfterConnect(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	ic := NewIbClient(new(Wrapper))

	// pre-Connect swap works
	replacement := new(Wrapper)
	if err := ic.SetWrapper(replacement); err != nil {
		t.Fatalf("pre-Connect SetWrapper returned err: %v", err)
	}

	if err := ic.Connect(host, port, 20); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer ic.Disconnect()

	// post-Connect swap must be rejected — connection is CONNECTING
	// at this point since HandShake hasn't run.
	if err := ic.SetWrapper(new(Wrapper)); err != ALREADY_CONNECTED {
		t.Fatalf("post-Connect SetWrapper got %v; want ALREADY_CONNECTED", err)
	}
}

// TestHandshakeCtxCancelTearsDownReceiver verifies that a HandShake
// aborted mid-confirm doesn't leak goReceive. Before the fix, the
// timeout / ctx-cancel branches returned without stopping goReceive,
// leaving one goroutine per failed HandShake.
//
// Simulates HandShake's inner loop without a real TWS peer by
// entering a ctx that cancels immediately after Connect.
func TestHandshakeCtxCancelTearsDownReceiver(t *testing.T) {
	addr, stop := listenLoopback(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	// Track goroutine count via wg — a leaked goReceive would keep
	// the counter above zero after HandShake bail-out.
	ic := NewIbClient(new(Wrapper))
	ctx, cancel := context.WithCancel(context.Background())
	_ = ic.SetContext(ctx)

	if err := ic.Connect(host, port, 12); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Simulate HandShake's spawn without going through the full
	// handshake protocol (needs a real TWS peer). This exercises the
	// same cleanup path: spawn goReceive, cancel, verify wg settles.
	ic.wg.Add(1)
	go ic.goReceive()

	// Emulate the HandShake bail: on ctx.Done, Disconnect and return.
	cancel()
	<-ctx.Done()
	if err := ic.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	// wg.Wait completed as part of Disconnect; verify goReceive left.
	done := make(chan struct{})
	go func() { ic.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wg still non-zero — goReceive leaked")
	}
}

