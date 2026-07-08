# ibapi — Go client for Interactive Brokers TWS/Gateway

Fork of [`github.com/hadrianl/ibapi`](https://github.com/hadrianl/ibapi)
at commit `4f647c0` (upstream 2023-09-27, unmaintained).

This fork lives at `github.com/fiber/ibapi`. Fixes land here first
before propagating to downstream consumers.

* Interactive Brokers TWS API 9.80
* Pure Go implementation of the binary protocol
* Unofficial — use at your own risk

## Install

```
go get github.com/fiber/ibapi
```

---

## Fork changelog

Consolidated list of changes relative to upstream `hadrianl/ibapi@4f647c0`.
Each bullet corresponds to a single commit; see `git log` for the
diff-level detail.

### Correctness

- **decoder: parse volume/size/position/filled fields as decimals.**
  Modern TWS emits volume, size, position, and filled quantities as
  decimals to support fractional shares. Upstream still read them as
  `int64` via `strconv.ParseInt`, panicking the receiver on any
  fractional value. Twelve decoder call sites switched to `readFloat`,
  wire types aligned with the official Python SDK's `Decimal` fields.
  `IbWrapper` signatures updated in lock-step (`TickSize.size`,
  `RealtimeBar.volume`, `PnlSingle.position`, `TickByTickAllLast.size`,
  `TickByTickBidAsk.bidSize/askSize`, `UpdateMktDepth[L2].size`).
  Struct fields updated: `RealTimeBar.Volume`, `HistogramData.Count`,
  `HistoricalTick.Size`, `HistoricalTickBidAsk.SizeBid/SizeAsk`,
  `HistoricalTickLast.Size`. Regression coverage in
  `TestDecoderFractionalSizes`.

- **client: close `done` channel via `sync.Once`, broadcast to all
  receivers.** Upstream's `Disconnect` guarded the done-signal with
  `if len(ic.done) > 0 { ic.done <- true }`, which is always false on
  an unbuffered channel — so `LoopUntilDone` hung forever after a
  clean shutdown. Replaced with `sync.Once` + `close(done)` for a
  broadcast-safe, idempotent signal. Regression coverage in
  `TestDisconnectNoReceiver` and `TestDisconnectWithReceiver`.

- **client: stop re-spawning goroutines on panic; tear down instead.**
  `goRequest`, `goReceive`, and `goDecode` each recovered their own
  panics and spawned a replacement goroutine. Three problems: the
  recover assumed `.(string)` and would crash on any other panic type;
  `wg` counters drifted below zero as originals' `Done()` fired while
  replacements ran; and silent restarts hid broken wrappers. Replaced
  with graceful teardown — record error into `ic.err`, close
  `terminatedSignal` via a new `shutdownOnce`, exit through the normal
  `wg.Done()` path. Regression coverage in
  `TestDecoderPanicTearsDownConnection`,
  `TestPanicWithNonStringValue`, and `TestSignalShutdownIdempotent`.

- **decoder: replaced two dead-code `log.Panic` calls in `goReceive`'s
  scanner-error handling with plain `log.Error` + `ic.err` assignment.**
  With the panic-restart pattern gone these paths would have crashed
  the process. `Wrapper.Error` is still fired for `BAD_LENGTH` so
  callers get the diagnostic. Bundled with the goroutine fix above.

### API additions

- **`CancelCalculateImpliedVolatility(reqID)`.** Companion cancel for
  `CalculateImpliedVolatility`; upstream shipped the option-price
  pair but not the implied-vol one.

### Cleanups

- **`Order.Solictied` → `Solicited`.** Typo in the upstream field name
  (both the struct field and the two decoder sites that populated it).

### Deliberately not fixed (yet)

- **No auto-reconnect.** Socket close means the caller must drive
  reconnection. Belongs to the calling application, not to this
  library.
- **No backpressure on `msgChan`** (buffer size 100). No observed
  impact under normal ops; deferred until a real burst causes trouble.

### Known protocol gap vs the current Python SDK

Announced client version is `MIN_SERVER_VER_WSHE_CALENDAR` (161).
Python's current cap is `MIN_SERVER_VER_RFQ_FIELDS` (187). Nothing
breaks — TWS gates outgoing fields on the announced version — but
newer Order/Contract metadata (fractional-size increment metadata,
pegged-mid/best offsets, customer-account fields, fund fields,
`HISTORICAL_SCHEDULE`, `USER_INFO`, IB white-branding, etc.) is not
yet plumbed through. Adds happen on demand as consumers need them.

---

## Usage

Implement `IbWrapper` to handle messages delivered via TWS or Gateway.
The default `Wrapper` in this package logs each callback to stdout —
useful for early wiring, not for anything real.

1. Implement your own `IbWrapper`
2. `Connect` to TWS or Gateway
3. `HandShake` with TWS or Gateway
4. `Run` the loop
5. Fire requests

### Demo 1

```go
import (
    . "github.com/fiber/ibapi"
    "time"
)

func main() {
    // Internal API log is zap; use GetLogger to fetch the logger or
    // SetAPILogger to install your own config.
    log := GetLogger().Sugar()
    defer log.Sync()

    // Wrapper{} below is a default implementation that just logs the msg.
    ic := NewIbClient(&Wrapper{})

    // TCP connect. Will fail if TWS/Gateway hasn't allow-listed this IP.
    if err := ic.Connect("127.0.0.1", 4002, 0); err != nil {
        log.Panic("Connect failed:", err)
    }

    // Handshake: exchange version and receive server time. Fails if
    // another client is already connected with the same clientID.
    if err := ic.HandShake(); err != nil {
        log.Panic("HandShake failed:", err)
    }

    // Queue requests. Nothing is sent until Run().
    ic.ReqCurrentTime()
    ic.ReqAutoOpenOrders(true)
    ic.ReqAccountUpdates(true, "")
    ic.ReqExecutions(ic.GetReqID(), ExecutionFilter{})

    ic.Run()
    <-time.After(time.Second * 60)
    ic.Disconnect()
}
```

### Demo 2 with context

```go
import (
    . "github.com/fiber/ibapi"
    "time"
    "context"
)

func main() {
    var err error
    SetAPILogger(zap.NewDevelopmentConfig()) // default is production (json, info); dev is console + debug
    log := GetLogger().Sugar()
    defer log.Sync()

    ic := NewIbClient(&Wrapper{})
    ctx, _ := context.WithTimeout(context.Background(), time.Second*30)
    ic.SetContext(ctx)

    if err = ic.Connect("127.0.0.1", 4002, 0); err != nil {
        log.Panic("Connect failed:", err)
    }
    if err = ic.HandShake(); err != nil {
        log.Panic("HandShake failed:", err)
    }

    ic.ReqCurrentTime()
    ic.ReqAutoOpenOrders(true)
    ic.ReqAccountUpdates(true, "")
    ic.ReqExecutions(ic.GetReqID(), ExecutionFilter{})

    ic.Run()
    err = ic.LoopUntilDone() // blocks until ctx cancels or Disconnect fires
    log.Info(err)
}
```

---

## Reference

1. [Official TWS API documentation](https://interactivebrokers.github.io/tws-api/)
2. [Order Types](https://www.interactivebrokers.com/en/index.php?f=4985)
3. [Product](https://www.interactivebrokers.com/en/index.php?f=4599)
4. [Margin](https://www.interactivebrokers.com/en/index.php?f=24176)
5. [Market Data](https://www.interactivebrokers.com/en/index.php?f=14193)
6. [Commissions](https://www.interactivebrokers.com/en/index.php?f=1590)
