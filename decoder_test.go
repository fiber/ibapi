package ibapi

import (
	"fmt"
	"testing"
	"time"
)

var decoder = &ibDecoder{
	wrapper: &Wrapper{},
}

func init() {
	decoder.setVersion(151)
	decoder.setmsgID2process()
}

func TestDecodeLongName(t *testing.T) {
	longName := "\\xef"
	fmt.Println(longName)
}

// captureWrapper records the arguments of size/volume callbacks so decoder
// tests can assert on them without a live TWS.
type captureWrapper struct {
	Wrapper
	tickSize          func(reqID, tickType int64, size float64)
	realtimeBar       func(reqID, t int64, o, h, l, c, v, wap float64, count int64)
	tickByTickAllLast func(reqID, tickType, t int64, price, size float64)
	tickByTickBidAsk  func(reqID, t int64, bidPrice, askPrice, bidSize, askSize float64)
	updateMktDepth    func(reqID, position, operation, side int64, price, size float64)
	pnlSingle         func(reqID int64, position, dailyPnL, unrealizedPnL, realizedPnL, value float64)
	histogramData     func(reqID int64, hist []HistogramData)
}

func (w *captureWrapper) TickSize(reqID, tickType int64, size float64) {
	if w.tickSize != nil {
		w.tickSize(reqID, tickType, size)
	}
}

func (w *captureWrapper) RealtimeBar(reqID, t int64, o, h, l, c, v, wap float64, count int64) {
	if w.realtimeBar != nil {
		w.realtimeBar(reqID, t, o, h, l, c, v, wap, count)
	}
}

func (w *captureWrapper) TickByTickAllLast(reqID, tickType, t int64, price, size float64, _ TickAttribLast, _, _ string) {
	if w.tickByTickAllLast != nil {
		w.tickByTickAllLast(reqID, tickType, t, price, size)
	}
}

func (w *captureWrapper) TickByTickBidAsk(reqID, t int64, bidPrice, askPrice, bidSize, askSize float64, _ TickAttribBidAsk) {
	if w.tickByTickBidAsk != nil {
		w.tickByTickBidAsk(reqID, t, bidPrice, askPrice, bidSize, askSize)
	}
}

func (w *captureWrapper) UpdateMktDepth(reqID, position, operation, side int64, price, size float64) {
	if w.updateMktDepth != nil {
		w.updateMktDepth(reqID, position, operation, side, price, size)
	}
}

func (w *captureWrapper) PnlSingle(reqID int64, position, dailyPnL, unrealizedPnL, realizedPnL, value float64) {
	if w.pnlSingle != nil {
		w.pnlSingle(reqID, position, dailyPnL, unrealizedPnL, realizedPnL, value)
	}
}

func (w *captureWrapper) HistogramData(reqID int64, hist []HistogramData) {
	if w.histogramData != nil {
		w.histogramData(reqID, hist)
	}
}

// msgBytes builds a decoder input from string fields separated by NUL.
func msgBytes(fields ...string) []byte {
	var out []byte
	for _, f := range fields {
		out = append(out, []byte(f)...)
		out = append(out, 0)
	}
	return out
}

// TestDecoderFractionalSizes locks in the fix for fractional volume/size
// fields (bug #6). Regressing any of these to readInt() will crash the
// decoder on modern TWS payloads.
func TestDecoderFractionalSizes(t *testing.T) {
	newDecoder := func() *ibDecoder {
		d := &ibDecoder{wrapper: &Wrapper{}}
		d.setVersion(176)
		d.setmsgID2process()
		return d
	}

	t.Run("TICK_SIZE fractional", func(t *testing.T) {
		var got float64
		w := &captureWrapper{tickSize: func(_, _ int64, size float64) { got = size }}
		d := newDecoder()
		d.setWrapper(w)
		// mTICK_SIZE=2; fields: msgID, version, reqID, tickType, size
		d.interpret(msgBytes("2", "6", "11", "0", "0.5"))
		if got != 0.5 {
			t.Fatalf("TickSize got %v, want 0.5", got)
		}
	})

	t.Run("REAL_TIME_BARS fractional volume", func(t *testing.T) {
		var vol float64
		w := &captureWrapper{realtimeBar: func(_, _ int64, _, _, _, _, v, _ float64, _ int64) { vol = v }}
		d := newDecoder()
		d.setWrapper(w)
		// mREAL_TIME_BARS=50; fields: msgID, version, reqID, time, o, h, l, c, volume, wap, count
		d.interpret(msgBytes("50", "1", "42",
			fmt.Sprintf("%d", time.Now().Unix()),
			"100.0", "101.0", "99.5", "100.5",
			"12.5", "100.25", "7"))
		if vol != 12.5 {
			t.Fatalf("RealtimeBar volume got %v, want 12.5", vol)
		}
	})

	t.Run("TICK_BY_TICK all-last fractional size", func(t *testing.T) {
		var size float64
		w := &captureWrapper{tickByTickAllLast: func(_, _, _ int64, _, s float64) { size = s }}
		d := newDecoder()
		d.setWrapper(w)
		// mTICK_BY_TICK=99; fields: msgID, reqID, tickType, time, price, size, mask, exchange, spec
		d.interpret(msgBytes("99", "7", "1", "1700000000", "100.0", "0.75", "0", "NYSE", ""))
		if size != 0.75 {
			t.Fatalf("TickByTickAllLast size got %v, want 0.75", size)
		}
	})

	t.Run("TICK_BY_TICK bid-ask fractional sizes", func(t *testing.T) {
		var bs, as float64
		w := &captureWrapper{tickByTickBidAsk: func(_, _ int64, _, _, bidSize, askSize float64) {
			bs, as = bidSize, askSize
		}}
		d := newDecoder()
		d.setWrapper(w)
		d.interpret(msgBytes("99", "7", "3", "1700000000", "100.0", "100.1", "0.5", "1.5", "0"))
		if bs != 0.5 || as != 1.5 {
			t.Fatalf("TickByTickBidAsk got (bid=%v ask=%v), want (0.5, 1.5)", bs, as)
		}
	})

	t.Run("MARKET_DEPTH fractional size", func(t *testing.T) {
		var got float64
		w := &captureWrapper{updateMktDepth: func(_, _, _, _ int64, _, size float64) { got = size }}
		d := newDecoder()
		d.setWrapper(w)
		// mMARKET_DEPTH=12; fields: msgID, version, reqID, position, operation, side, price, size
		d.interpret(msgBytes("12", "1", "1", "0", "1", "1", "100.0", "0.25"))
		if got != 0.25 {
			t.Fatalf("UpdateMktDepth size got %v, want 0.25", got)
		}
	})

	t.Run("PNL_SINGLE fractional position", func(t *testing.T) {
		var pos float64
		w := &captureWrapper{pnlSingle: func(_ int64, position, _, _, _, _ float64) { pos = position }}
		d := newDecoder()
		d.setWrapper(w)
		// mPNL_SINGLE=95; fields: msgID, reqID, position, dailyPnL, unrealizedPnL, realizedPnL, value
		d.interpret(msgBytes("95", "1", "0.3333", "0", "0", "0", "0"))
		if pos != 0.3333 {
			t.Fatalf("PnlSingle position got %v, want 0.3333", pos)
		}
	})

	t.Run("HISTOGRAM_DATA fractional count", func(t *testing.T) {
		var hist []HistogramData
		w := &captureWrapper{histogramData: func(_ int64, h []HistogramData) { hist = h }}
		d := newDecoder()
		d.setWrapper(w)
		// mHISTOGRAM_DATA=89; fields: msgID, reqID, n, price1, count1, price2, count2
		d.interpret(msgBytes("89", "1", "2", "100.0", "0.5", "101.0", "2.25"))
		if len(hist) != 2 || hist[0].Count != 0.5 || hist[1].Count != 2.25 {
			t.Fatalf("HistogramData got %+v, want counts 0.5 and 2.25", hist)
		}
	})
}

func BenchmarkDecode(b *testing.B) {
	// log, _ = zap.NewDevelopment()
	msgUpdateAccountValue := []byte{54, 0, 50, 0, 78, 101, 116, 76, 105, 113, 117, 105, 100, 97, 116, 105, 111, 110, 66, 121, 67, 117, 114, 114, 101, 110, 99, 121, 0, 45, 49, 49, 48, 53, 54, 49, 50, 0, 72, 75, 68, 0, 68, 85, 49, 51, 56, 50, 56, 51, 55, 0}
	msgHistoricalDataUpdate := []byte{57, 48, 0, 50, 0, 50, 48, 57, 0, 50, 48, 50, 48, 48, 53, 50, 54, 32, 32, 49, 54, 58, 50, 48, 58, 48, 48, 0, 50, 51, 52, 48, 51, 0, 50, 51, 52, 48, 52, 0, 50, 51, 52, 48, 54, 0, 50, 51, 52, 48, 48, 0, 50, 51, 52, 48, 51, 46, 52, 51, 56, 49, 54, 50, 53, 52, 52, 49, 55, 0, 50, 56, 51, 0}
	msgUpdateMktDepthL2 := []byte{49, 51, 0, 49, 0, 51, 0, 48, 0, 72, 75, 70, 69, 0, 49, 0, 49, 0, 50, 51, 52, 48, 51, 0, 56, 0, 49, 0}
	msgError := []byte{52, 0, 50, 0, 45, 49, 0, 50, 49, 48, 54, 0, 72, 77, 68, 83, 32, 100, 97, 116, 97, 32, 102, 97, 114, 109, 32, 99, 111, 110, 110, 101, 99, 116, 105, 111, 110, 32, 105, 115, 32, 79, 75, 58, 102, 117, 110, 100, 102, 97, 114, 109, 0}
	// var updateAccountValueMsgBuf = NewMsgBuffer(nil)
	for i := 0; i < b.N; i++ {
		// updateAccountValueMsgBuf.Write(msgBytes)
		decoder.interpret(msgUpdateAccountValue)
		decoder.interpret(msgHistoricalDataUpdate)
		decoder.interpret(msgUpdateMktDepthL2)
		decoder.interpret(msgError)
	}
}
