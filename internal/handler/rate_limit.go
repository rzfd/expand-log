package handler

import (
	"sync"
	"time"
)

type slidingWindowLimiter struct {
	mu     sync.Mutex
	window time.Duration
	limit  int
	hits   map[string][]time.Time
}

func newSlidingWindowLimiter(window time.Duration, limit int) *slidingWindowLimiter {
	return &slidingWindowLimiter{
		window: window,
		limit:  limit,
		hits:   make(map[string][]time.Time),
	}
}

func (l *slidingWindowLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	events := l.hits[key]
	cutoff := now.Add(-l.window)
	kept := events[:0]
	for _, ts := range events {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	events = kept

	if len(events) >= l.limit {
		retryAfter := l.window - now.Sub(events[0])
		if retryAfter < 0 {
			retryAfter = 0
		}
		l.hits[key] = events
		return false, retryAfter
	}

	events = append(events, now)
	l.hits[key] = events
	return true, 0
}

func (l *slidingWindowLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
