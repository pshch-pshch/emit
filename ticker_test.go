package emit_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/pshch-pshch/emit"
)

func TestTicker_Period(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ticker period check in short mode")
	}

	const n = 1000
	period := 1 * time.Millisecond
	ticker := emit.NewTicker(period)
	defer ticker.Stop()

	t0 := time.Now()
	for i := 0; i < n; i++ {
		<-ticker.C
	}
	t1 := time.Now()
	dt := t1.Sub(t0)

	expected := period * n
	slop := expected * 2 / 10
	if dt < expected-slop || dt > expected+slop {
		t.Fatalf("%d %s ticks took %s, expected [%s,%s]", n, period, dt, expected-slop, expected+slop)
	}
}

func TestTicker_Reset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reset ticker period check in short mode")
	}

	const n = 1000
	period := 1 * time.Millisecond
	ticker := emit.NewTicker(2 * period) // Start Ticker with double period
	defer ticker.Stop()

	<-ticker.C
	ticker.Reset(period) // Reset to expected period now

	t0 := time.Now()
	for i := 0; i < n; i++ {
		<-ticker.C
	}
	t1 := time.Now()
	dt := t1.Sub(t0)

	expected := period * n
	slop := expected * 2 / 10
	if dt < expected-slop || dt > expected+slop {
		t.Fatalf("%d %s ticks took %s, expected [%s,%s]", n, period, dt, expected-slop, expected+slop)
	}
}

func TestTicker_Skip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ticker skip check in short mode")
	}

	const n = 1000
	period := 1 * time.Millisecond
	ticker := emit.NewTicker(period)
	defer ticker.Stop()

	t0 := <-ticker.C
	time.Sleep(period * n)
	t1 := <-ticker.C
	dt := t1.Sub(t0)

	expected := period * n
	slop := expected * 2 / 10
	if dt < expected-slop || dt > expected+slop {
		t.Fatalf("%d skipped %s ticks interval is %s, expected [%s,%s]", n, period, dt, expected-slop, expected+slop)
	}
}

func TestTicker_Stop(t *testing.T) {
	period := 1 * time.Millisecond
	ticker := emit.NewTicker(period)

	<-ticker.C
	ticker.Stop()

	select {
	case <-ticker.C:
		t.Fatal("Can receive from stopped ticker")
	case <-time.After(2 * period):
	}
}

func TestTicker_Pause(t *testing.T) {
	period := 1 * time.Millisecond
	ticker := emit.NewTicker(period)
	defer ticker.Stop()

	<-ticker.C
	ticker.Reset(0) // Pause Ticker

	select {
	case <-ticker.C:
		t.Fatal("Can receive from paused ticker")
	case <-time.After(2 * period):
	}

	ticker.Reset(period) // Resume Ticker
	<-ticker.C
}

func TestTicker_CloseOnStop(t *testing.T) {
	period := 1 * time.Millisecond
	ticker := emit.TickerConfig{
		CloseOnStop: true,
	}.NewTicker(period)

	<-ticker.C
	ticker.Stop()

	if _, ok := <-ticker.C; ok {
		t.Fatal("Ticker channel is not closed")
	}
}

func TestTicker_DropOnReset(t *testing.T) {
	period := 1 * time.Millisecond
	ticker := emit.TickerConfig{
		DropTickOnReset: true,
	}.NewTicker(period)
	defer ticker.Stop()

	time.Sleep(2 * period)
	ticker.Reset(period)

	runtime.Gosched()
	select {
	case <-ticker.C:
		t.Fatal("Can receive from reset ticker")
	default:
	}
}

func TestTicker_KeepOnReset(t *testing.T) {
	period := 1 * time.Millisecond
	ticker := emit.TickerConfig{
		DropTickOnReset: false,
	}.NewTicker(period)
	defer ticker.Stop()

	time.Sleep(2 * period)
	ticker.Reset(period)

	select {
	case <-ticker.C:
	case <-time.After(period / 2):
		t.Fatal("Can't receive from reset ticker")
	}
}

func TestTicker_DropOnStop(t *testing.T) {
	period := 1 * time.Millisecond
	ticker := emit.TickerConfig{
		DropTickOnStop: true,
	}.NewTicker(period)

	time.Sleep(2 * period)
	ticker.Stop()

	runtime.Gosched()
	select {
	case <-ticker.C:
		t.Fatal("Can receive from stopped ticker")
	default:
	}
}

func TestTicker_KeepOnStop(t *testing.T) {
	period := 1 * time.Millisecond
	ticker := emit.TickerConfig{
		DropTickOnStop: false,
	}.NewTicker(period)

	time.Sleep(2 * period)
	ticker.Stop()

	select {
	case <-ticker.C:
	case <-time.After(period / 2):
		t.Fatal("Can't receive from stopped ticker")
	}
}
