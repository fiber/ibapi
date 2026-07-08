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

## Why this fork?

Upstream is unmaintained since 2023-09 and carries a few load-bearing
bugs that block real trading:

- The decoder panics on fractional share volumes / sizes / positions
  because modern TWS emits those fields as decimals.
- `Disconnect()` never signals `LoopUntilDone` — the shutdown-done
  guard is a `len(chan) > 0` check that's always false.
- The three worker goroutines catch their own panics and try to
  restart themselves, leaking the original and hiding broken wrappers.

This fork fixes all three, plus a handful of drive-by cleanups. See
[CHANGELOG.md](CHANGELOG.md) for per-commit detail and the known
protocol-version gap vs the current Python SDK.

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
