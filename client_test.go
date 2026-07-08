package ibapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// devLogger returns a text-format slog.Logger at debug level — the
// equivalent of the previous zap.NewDevelopmentConfig used by these
// live-TWS smoke tests.
func devLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestClient(t *testing.T) {
	SetLogger(devLogger())
	runtime.GOMAXPROCS(4)

	ic := NewIbClient(new(Wrapper))
	_ = ic.SetConnectionOptions("+PACEAPI")

	if err := ic.Connect("localhost", 7497, 100); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	if err := ic.HandShake(); err != nil {
		t.Fatalf("failed to hand shake: %v", err)
	}
	ic.Run()

	// ####################### request base info ##################################################################
	ic.ReqCurrentTime()
	ic.ReqAutoOpenOrders(true)
	ic.ReqAccountUpdates(true, "")
	ic.ReqExecutions(ic.GetReqID(), ExecutionFilter{})
	ic.ReqAllOpenOrders()
	ic.ReqPositions()
	ic.ReqCompletedOrders(false)

	ic.LoopUntilDone(
		func() {
			<-time.After(time.Second * 25)
			ic.Disconnect()
		})
}

func TestClientReconnect(t *testing.T) {
	SetLogger(devLogger())
	runtime.GOMAXPROCS(4)

	ic := NewIbClient(new(Wrapper))

	for {
		_ = ic.SetConnectionOptions("+PACEAPI")
		if err := ic.Connect("localhost", 4002, 0); err != nil {
			log.Error("failed to connect, reconnect after 5 sec", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if err := ic.HandShake(); err != nil {
			log.Error("failed to hand shake, reconnect after 5 sec", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}
		ic.Run()
		ic.LoopUntilDone(func() {
			<-time.After(25 * time.Second)
			ic.Disconnect()
		})
	}
}

func TestClientWithContext(t *testing.T) {
	runtime.GOMAXPROCS(4)
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30000)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ibwrapper := new(Wrapper)
	ic := NewIbClient(ibwrapper)
	_ = ic.SetContext(ctx)
	_ = ic.SetConnectionOptions("+PACEAPI")
	err = ic.Connect("localhost", 7497, 0)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	err = ic.HandShake()
	if err != nil {
		t.Fatalf("failed to hand shake: %v", err)
	}
	ic.Run()

	ic.ReqCurrentTime()
	ic.ReqAccountUpdates(true, "")

	hsi := Contract{ContractID: 500007591, Symbol: "HSI", SecurityType: "FUT", Exchange: "HKFE"}
	ic.ReqContractDetails(ic.GetReqID(), &hsi)
	ic.ReqMktData(ic.GetReqID(), &hsi, "", false, false, nil)

	tags := []string{"AccountType", "NetLiquidation", "TotalCashValue", "SettledCash",
		"AccruedCash", "BuyingPower", "EquityWithLoanValue",
		"PreviousEquityWithLoanValue", "GrossPositionValue", "ReqTEquity",
		"ReqTMargin", "SMA", "InitMarginReq", "MaintMarginReq", "AvailableFunds",
		"ExcessLiquidity", "Cushion", "FullInitMarginReq", "FullMaintMarginReq",
		"FullAvailableFunds", "FullExcessLiquidity", "LookAheadNextChange",
		"LookAheadInitMarginReq", "LookAheadMaintMarginReq",
		"LookAheadAvailableFunds", "LookAheadExcessLiquidity",
		"HighestSeverity", "DayTradesRemaining", "Leverage", "$LEDGER:ALL"}
	ic.ReqAccountSummary(ic.GetReqID(), "All", strings.Join(tags, ","))

	ic.ReqHistoricalData(ic.GetReqID(), &hsi, "", "4800 S", "1 min", "TRADES", false, 1, true, nil)

	pprofServe := func() {
		http.ListenAndServe("localhost:6060", nil)
	}

	go pprofServe()

	f := func() {
		sig := <-sigs
		fmt.Print(sig)
		cancel()
	}

	err = ic.LoopUntilDone(f)
	fmt.Println(err)
}

func BenchmarkPlaceOrder(b *testing.B) {
	ibwrapper := new(Wrapper)
	ic := NewIbClient(ibwrapper)
	ic.setConnState(2)
	ic.serverVersion = 151
	contract := new(Contract)
	order := new(Order)

	go func() {
		for {
			<-ic.reqChan
		}
	}()

	for i := 0; i < b.N; i++ {
		ic.PlaceOrder(1, contract, order)
	}
}

func BenchmarkAppendEmptySlice(b *testing.B) {
	arr := []byte("benchmark test of append and copy")
	for i := 0; i < b.N; i++ {
		_ = append([]byte{}, arr...)
	}
}

func BenchmarkCopySlice(b *testing.B) {
	arr := []byte("benchmark test of append and copy")
	for i := 0; i < b.N; i++ {
		oarr := arr
		newSlice := make([]byte, len(oarr))
		copy(newSlice, oarr)
		_ = newSlice
	}
}

func TestPlaceOrder(t *testing.T) {
	SetLogger(devLogger())
	runtime.GOMAXPROCS(4)
	var err error

	ibwrapper := new(Wrapper)
	ic := NewIbClient(ibwrapper)
	_ = ic.SetConnectionOptions("+PACEAPI")

	err = ic.Connect("localhost", 7497, 0)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	err = ic.HandShake()
	if err != nil {
		t.Fatalf("failed to hand shake: %v", err)
	}
	ic.Run()

	ic.ReqCurrentTime()
	ic.ReqAccountUpdates(true, "")

	aapl := Contract{ContractID: 265598, Symbol: "AAPL", SecurityType: "STK", Exchange: "NYSE"}
	ic.ReqContractDetails(ic.GetReqID(), &aapl)
	ic.ReqMktData(ic.GetReqID(), &aapl, "", false, false, nil)

	lmtOrder := NewLimitOrder("BUY", 144, 1)
	ic.PlaceOrder(ibwrapper.GetNextOrderID(), &aapl, lmtOrder)

	ic.LoopUntilDone(
		func() {
			<-time.After(time.Second * 25)
			ic.Disconnect()
		})

	fmt.Println(err)
}
