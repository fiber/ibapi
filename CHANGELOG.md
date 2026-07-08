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

### Added

- **`CancelCalculateImpliedVolatility(reqID)`.** Companion cancel for
  `CalculateImpliedVolatility`; upstream shipped the option-price
  pair but not the implied-vol one.

### Changed

- **Module path**: `github.com/hadrianl/ibapi` → `github.com/fiber/ibapi`.
- **`Order.Solictied` → `Solicited`.** Typo in the upstream field name
  (both the struct field and the two decoder sites that populated it).

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
