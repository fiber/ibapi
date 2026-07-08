# Changelog

All notable changes in this fork relative to upstream
[`github.com/hadrianl/ibapi`](https://github.com/hadrianl/ibapi) at
commit `4f647c0` (2023-09-27, unmaintained). Each bullet corresponds
to a single commit; see `git log` for diff-level detail.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Fixed

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

- **client: end-to-end `Disconnect` idempotency + `wg.Add`-before-`go`.**
  Two related lifecycle races. First: `Disconnect` was only guarded
  at the channel-close level, not the whole body, so concurrent calls
  (e.g. panicking-worker-driven Disconnect racing a user-driven one)
  raced through `reset()`'s ~15 field writes. Wrapped body in a new
  `disconnectOnce`. Second: each worker did `wg.Add(1)` inside itself,
  letting a fast `Disconnect` hit `wg.Wait()` with counter 0 before
  the worker scheduled. Moved the `Add` to the caller side.
  Regression coverage in `TestConcurrentDisconnect` and
  `TestFastRunThenDisconnect` — both meaningful only under `-race`.

- **client/connection: atomic-guard `ic.err`, `conn.state`, byte
  counters.** Three previously-unsynchronized fields. `ic.err` (a
  two-word interface) was written from four sites across three
  workers; torn writes were possible on simultaneous socket failures.
  Add a `setErr` helper guarded by `errOnce` — first-error-wins matches
  the "causative error" semantic. `IbConnection.state` (read by
  `IsConnected` from `goRequest`, written by `HandShake` / `Disconnect`
  from the main goroutine) → `atomic.Int32`; state constants retyped
  to `int32` iota. `numBytesSent/Recv` and `numMsgSent/Recv`
  (incremented on every socket op from `goRequest` / `goReceive`) →
  `atomic.Int64`. Regression coverage in `TestSetErrFirstWins`.

- **client: harden `HandShake`, request enqueue, `LoopUntilDone`,
  nil-`Disconnect`.** Four API-boundary hazards. `HandShake` timeout /
  ctx-cancel branches returned without stopping the already-spawned
  `goReceive` — now `defer` a `Disconnect()` on the non-OK path so
  the socket closes and the receiver actually exits. Also swapped
  `time.After` for `time.NewTimer` + `Stop`. `PlaceOrder`-style
  callers sending on `reqChan` after `Disconnect` blocked forever
  once the 10-slot buffer filled — introduce an `enqueue()` helper
  that checks `IsConnected` atomically and `select`s against
  `terminatedSignal`; 78 direct sends rewritten. `LoopUntilDone`'s
  ctx-watcher goroutine leaked when `Disconnect` fired directly (not
  via ctx cancel) — added a `<-ic.done` branch. `Disconnect` before
  `Connect` nil-derefed on the zero-value `*net.TCPConn` — nil-guard
  in `IbConnection.disconnect()`. Also fixes a defer-order race the
  new tests surfaced: workers had `wg.Done()` registered last, so it
  fired first at exit and `Disconnect`'s deferred `reset()` swapped
  channels while the recover was still reading `terminatedSignal`.
  Reordered so `wg.Done` fires last. Regression coverage in
  `TestDisconnectBeforeConnect`, `TestRequestAfterDisconnectDoesNotHang`,
  `TestLoopUntilDoneWatcherReleasedOnDirectDisconnect`,
  `TestHandshakeCtxCancelTearsDownReceiver`.

- **client: reject setters after `Connect` (Python-model contract).**
  `SetWrapper`, `SetContext`, `SetConnectionOptions` all mutated
  fields readable from HandShake / worker goroutines. Match Python's
  construction-time contract — the wrapper is passed at
  `NewIbClient(wrapper)` and setters must be called before `Connect`.
  Post-Connect calls return `ALREADY_CONNECTED` instead of racing
  the readers. Signature change: all three setters now return
  `error`. Regression coverage in `TestSettersRejectedAfterConnect`
  and `TestSetWrapperRejectedAfterConnect`.

### Added

- **`CancelCalculateImpliedVolatility(reqID)`.** Companion cancel for
  `CalculateImpliedVolatility`; upstream shipped the option-price
  pair but not the implied-vol one.
- **`Option` + `WithLogger(*slog.Logger)`.** Functional-options
  pattern on `NewIbClient(wrapper, opts ...Option)` for
  construction-time configuration. Coexists with the imperative
  Python-style setters — callers choose whichever style fits.

### Changed

- **Module path**: `github.com/hadrianl/ibapi` → `github.com/fiber/ibapi`.
- **`Order.Solictied` → `Solicited`.** Typo in the upstream field name
  (both the struct field and the two decoder sites that populated it).
- **Logger: `go.uber.org/zap` → stdlib `log/slog`.** Removes the only
  external dependency in the fork. Package-scoped `log *slog.Logger`
  defaults to `slog.Default()`. Public API replacements:
  `SetAPILogger(zap.Config, ...zap.Option) error` → `SetLogger(*slog.Logger)`
  and `GetLogger() *zap.Logger` → `Logger() *slog.Logger`. All ~250
  `zap.X("k", v)` structured-field calls rewritten to slog's
  key-value varargs form. Sixteen production `log.Panic` sites (slog
  has no `Panic`) rewritten as `log.Error(...) + panic(value)` — the
  panic value is the underlying error where available, otherwise a
  descriptive string; existing worker-recover paths catch these the
  same way. See panic report below.

### Removed

- **Dead `IbClient.timeChan chan time.Time` field.** Declared,
  initialized nowhere, never read.
- **`go.uber.org/zap` dependency.** `go.mod` now lists zero external
  requires; `go.sum` is empty. Go floor bumped to 1.22 for `log/slog`.

## Panic sites (post slog migration)

Every remaining production `panic()` call in this package is caught by
one of the worker goroutines' recover blocks (`goRequest`, `goReceive`,
`goDecode`), converted to an `ic.err` value via `panicError`, and
followed by `signalShutdown()` — see the panic-restart fix. In other
words: a panic here tears down the connection cleanly rather than
crashing the process.

- `utils.go MsgBuffer.readInt / readIntCheckUnset / readFloat /
  readFloatCheckUnset / readBool / readString` — panic on
  `bytes.Buffer.ReadBytes` failure or `strconv.ParseInt/Float`
  failure. Effectively "malformed protocol message from TWS."
- `utils.go decodeInt` — same shape; unused by the current decoder
  but left for API compatibility.
- `utils.go makeMsgBytes` — panics if a caller passes a field type
  the encoder doesn't handle. Programmer error, not runtime data.
- `utils.go handleEmpty` — panics on any type other than int64/float64.
  Programmer error.
- `utils.go InitDefault` — panics on an unrecognized struct tag
  value. Programmer error (typo'd tag).
- `decoder.go processUpdateAccountTime` — panics on `time.Parse`
  failure. TWS protocol violation.
- `client.go PlaceOrder / ReqMktData / ReqMktDepth` — two "not
  supported" panics on `mktDataOptions` / `mktDepthOptions` slices
  documented as internal-use only. Panics only if a caller passes a
  non-empty slice.
- `orderCondition.go decodeCondition` — panics on unknown condition
  type. Wire-format violation or missing case in the switch.

## Deliberately not fixed (yet)

- **No auto-reconnect.** Socket close means the caller must drive
  reconnection. Belongs to the calling application, not to this
  library.
- **No backpressure on `msgChan`** (buffer size 100). No observed
  impact under normal ops; deferred until a real burst causes trouble.

## Known protocol gap vs the current Python SDK

Announced client version is `MIN_SERVER_VER_WSHE_CALENDAR` (161).
Python's current cap is `MIN_SERVER_VER_RFQ_FIELDS` (187). Nothing
breaks — TWS gates outgoing fields on the announced version — but
newer Order/Contract metadata (fractional-size increment metadata,
pegged-mid/best offsets, customer-account fields, fund fields,
`HISTORICAL_SCHEDULE`, `USER_INFO`, IB white-branding, etc.) is not
yet plumbed through. Adds happen on demand as consumers need them.
