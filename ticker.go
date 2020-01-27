package emit

import (
	"sync"
	"time"

	"github.com/pshch-pshch/chia"
)

// Ticker is an extended version of time.Ticker and behaves almost identical.
// Read standard library documentation as well.
// It can drop ticks to make up for slow receivers: only the latest skipped tick will be sent.
type Ticker struct {
	// The channel on which the ticks are delivered.
	C <-chan time.Time

	c chan time.Time

	cfg TickerConfig

	stop   *chia.Shutdown
	reset  chan tickerReset
	ticker *time.Ticker
}

type tickerReset struct {
	d    time.Duration
	done func()
}

// NewTicker creates a new Ticker with default TickerConfig and provided tick interval.
// Unlike in time.NewTicker duration can be zero, which leads to paused ticker that can be reset later.
func NewTicker(d time.Duration) *Ticker {
	return TickerConfig{}.NewTicker(d)
}

// TickerConfig allows Ticker startup customization.
type TickerConfig struct {
	// CloseOnStop determines if ticks channel will be closed on Ticker stop (e.g. to range over it).
	CloseOnStop bool
	// DropTickOnReset determines if unconsumed tick will be dropped on Reset.
	DropTickOnReset bool
	// DropTickOnStop determines if unconsumed tick will be dropped on Stop.
	DropTickOnStop bool
}

// NewTicker creates Ticker customized by TickerConfig. See TickerConfig description for details.
func (cfg TickerConfig) NewTicker(d time.Duration) *Ticker {
	c := make(chan time.Time, 1)

	t := &Ticker{
		C: c, c: c,

		cfg: cfg,
	}

	t.stop = chia.NewShutdown()
	t.reset = make(chan tickerReset)

	t.newTicker(d)

	go t.run()

	return t
}

// Reset behaves almost like stopping the Ticker and creating a new one with another period,
// but it keeps the same Ticker with the same channel for ticks delivering.
// Zero duration will cause Ticker to pause.
// Already stopped Ticker will not be altered (Reset is no-op in that case).
func (t *Ticker) Reset(d time.Duration) {
	var wg sync.WaitGroup
	wg.Add(1)

	select {
	case <-t.stop.Done:
	case t.reset <- tickerReset{d, wg.Done}:
		wg.Wait()
	}
}

// Stop turns off a ticker. After Stop, no more ticks will be sent.
// Unlike time.Ticker.Stop, channel may be closed depending on TickerConfig.CloseOnStop.
func (t *Ticker) Stop() {
	t.stop.CloseAndWait()
}

func (t *Ticker) run() {
	for {
		// Fast path for stop
		select {
		case done := <-t.stop.Init:
			t.handleStop(done)
			return
		default:
		}

		switch t.ticker {
		case nil:
			// Just wait for stop or reset
			select {
			case done := <-t.stop.Init:
				t.handleStop(done)
				return
			case r := <-t.reset:
				t.handleReset(r.d, r.done)
			}
		default:
			// Wait for the next upstream tick
			select {
			case done := <-t.stop.Init:
				t.handleStop(done)
				return
			case r := <-t.reset:
				t.handleReset(r.d, r.done)
			case tick := <-t.ticker.C:
				t.drain()
				t.c <- tick
			}
		}
	}
}

func (t *Ticker) handleStop(done func()) {
	if t.cfg.DropTickOnStop {
		t.drain()
	}
	if t.cfg.CloseOnStop {
		close(t.c)
	}
	t.newTicker(0)

	done()
}

func (t *Ticker) handleReset(d time.Duration, done func()) {
	if t.cfg.DropTickOnReset {
		t.drain()
	}
	t.newTicker(d)

	done()
}

// drain drops unconsumed buffered tick if any
func (t *Ticker) drain() {
	select {
	case <-t.c:
	default:
	}
}

// newTicker (re)creates internal time.Ticker
func (t *Ticker) newTicker(d time.Duration) {
	if t.ticker != nil {
		t.ticker.Stop()
	}

	if d == 0 {
		t.ticker = nil
	} else {
		t.ticker = time.NewTicker(d)
	}
}
