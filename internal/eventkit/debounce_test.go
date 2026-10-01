package eventkit

import (
	"context"
	"testing"
	"time"
)

func TestDebounceCoalescesBursts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan struct{}, 10)
	out := debounce(ctx, in, 30*time.Millisecond)
	for range 5 {
		in <- struct{}{}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case <-out:
	case <-time.After(time.Second):
		t.Fatal("no debounced signal")
	}
	select {
	case <-out:
		t.Fatal("burst produced more than one signal")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDebounceClosesOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out := debounce(ctx, make(chan struct{}), 10*time.Millisecond)
	cancel()
	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("expected close")
		}
	case <-time.After(time.Second):
		t.Fatal("not closed")
	}
}

func TestWithPollForwardsInputAndTicks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan struct{}, 1)
	out := withPoll(ctx, in, 20*time.Millisecond)
	in <- struct{}{}
	for range 2 { // one forwarded signal, then at least one tick
		select {
		case <-out:
		case <-time.After(time.Second):
			t.Fatal("expected signal")
		}
	}
}

func TestWithPollClosesOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out := withPoll(ctx, make(chan struct{}), time.Hour)
	cancel()
	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("expected close")
		}
	case <-time.After(time.Second):
		t.Fatal("not closed")
	}
}
