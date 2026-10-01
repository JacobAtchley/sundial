package eventkit

import (
	"context"
	"time"
)

// debounce emits once after in has been quiet for d. out closes when ctx
// is done or in closes.
func debounce(ctx context.Context, in <-chan struct{}, d time.Duration) <-chan struct{} {
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		timer := time.NewTimer(d)
		timer.Stop()
		var fire <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case _, ok := <-in:
				if !ok {
					return
				}
				timer.Reset(d)
				fire = timer.C
			case <-fire:
				fire = nil
				select {
				case out <- struct{}{}:
				default:
				}
			}
		}
	}()
	return out
}

// withPoll forwards every signal from in and also emits one each interval, so
// consumers still refresh when EventKit delivers no change notifications. out
// closes when ctx is done or in closes.
func withPoll(ctx context.Context, in <-chan struct{}, interval time.Duration) <-chan struct{} {
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-in:
				if !ok {
					return
				}
			case <-ticker.C:
			}
			select {
			case out <- struct{}{}:
			default:
			}
		}
	}()
	return out
}
