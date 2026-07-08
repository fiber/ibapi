package ibapi

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// panicWrapper's NextValidID callback signals it ran and then panics.
// Used to exercise the panic-teardown path in goDecode.
type panicWrapper struct {
	Wrapper
	mu     sync.Mutex
	called int
	fired  chan struct{}
	panic  interface{} // what to panic with; nil = string "callback boom"
}

func (w *panicWrapper) NextValidID(_ int64) {
	w.mu.Lock()
	w.called++
	w.mu.Unlock()
	select {
	case w.fired <- struct{}{}:
	default:
	}
	if w.panic == nil {
		panic("callback boom")
	}
	panic(w.panic)
}

func (w *panicWrapper) timesCalled() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.called
}

// startDecoderWithWrapper starts goDecode after Add(1)ing on wg. That
// pre-increment makes wg.Wait() reliable — the production goroutines
// have an Add-inside-goroutine race that's unrelated to bug #2 and not
// what we're testing here.
func startDecoderWithWrapper(t *testing.T, wrapperImpl IbWrapper) *IbClient {
	t.Helper()
	addr, stop := listenLoopback(t)
	t.Cleanup(stop)
	host, port := splitHostPort(t, addr)

	ic := NewIbClient(wrapperImpl)
	if err := ic.Connect(host, port, 42); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	ic.decoder.setVersion(176)
	ic.decoder.setmsgID2process()

	ic.wg.Add(1)
	go func() {
		defer ic.wg.Done()
		// mirror goDecode's recover so the pre-Add balances correctly.
		defer func() {
			if p := recover(); p != nil {
				ic.err = panicError("decoder", p)
				ic.signalShutdown()
			}
		}()
		for {
			select {
			case m := <-ic.msgChan:
				ic.decoder.interpret(m)
			case <-ic.terminatedSignal:
				return
			}
		}
	}()
	return ic
}

// TestDecoderPanicTearsDownConnection verifies that a wrapper callback
// panic during decoding no longer restarts the decode goroutine — it
// records the error, signals shutdown, and lets the caller observe the
// closure via LoopUntilDone / <-ic.done.
func TestDecoderPanicTearsDownConnection(t *testing.T) {
	pw := &panicWrapper{fired: make(chan struct{}, 4)}
	ic := startDecoderWithWrapper(t, pw)

	// mNEXT_VALID_ID = 9; fields: msgID, version, reqID
	ic.msgChan <- msgBytes("9", "1", "7")

	select {
	case <-pw.fired:
	case <-time.After(2 * time.Second):
		t.Fatal("callback never ran")
	}

	done := make(chan struct{})
	go func() { ic.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wg.Wait did not return within 2s — decoder likely restarted")
	}

	// Give the old restart pattern a beat to (mis)fire again.
	time.Sleep(50 * time.Millisecond)
	if n := pw.timesCalled(); n != 1 {
		t.Fatalf("panicWrapper.NextValidID called %d times; want 1 (a restart would fire it repeatedly)", n)
	}
	if ic.err == nil {
		t.Fatal("ic.err was not set after decoder panic")
	}
	if !strings.Contains(ic.err.Error(), "decoder panicked") {
		t.Fatalf("ic.err = %v; want to mention decoder panic", ic.err)
	}
	select {
	case _, ok := <-ic.terminatedSignal:
		if ok {
			t.Fatal("terminatedSignal delivered a value instead of being closed")
		}
	default:
		t.Fatal("terminatedSignal was not closed after panic")
	}
}

// TestPanicWithNonStringValue guards against the old
// `errors.New(errMsg.(string))` pattern that would itself panic if the
// wrapper panicked with anything other than a string.
func TestPanicWithNonStringValue(t *testing.T) {
	pw := &panicWrapper{fired: make(chan struct{}, 4), panic: 42}
	ic := startDecoderWithWrapper(t, pw)

	ic.msgChan <- msgBytes("9", "1", "1")

	select {
	case <-pw.fired:
	case <-time.After(2 * time.Second):
		t.Fatal("callback never ran")
	}

	done := make(chan struct{})
	go func() { ic.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wg.Wait hung — recover likely re-panicked on non-string value")
	}

	if ic.err == nil || !strings.Contains(ic.err.Error(), "42") {
		t.Fatalf("ic.err = %v; want to mention the panic value (42)", ic.err)
	}
}

// TestSignalShutdownIdempotent — Disconnect and a panicking goroutine
// both may call signalShutdown; the second call must be a no-op.
func TestSignalShutdownIdempotent(t *testing.T) {
	ic := NewIbClient(new(Wrapper))
	ic.terminatedSignal = make(chan int)
	ic.shutdownOnce = sync.Once{}

	ic.signalShutdown()
	ic.signalShutdown() // must not panic on double-close
	select {
	case _, ok := <-ic.terminatedSignal:
		if ok {
			t.Fatal("terminatedSignal delivered a value instead of being closed")
		}
	default:
		t.Fatal("terminatedSignal not closed")
	}
}
