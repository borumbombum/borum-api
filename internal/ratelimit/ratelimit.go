// Package ratelimit provides a small in-memory sliding-window rate limiter
// keyed per client. A per-process counter is plenty for this single-user app;
// it exists only to slow brute-force attempts and abuse of public endpoints.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter bounds how many attempts fit inside a rolling window, keyed by any
// string (client IP, client IP + email, ...). Idle keys are pruned so the map
// cannot grow without bound.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string]*entry
}

type entry struct {
	times    []time.Time
	lastSeen time.Time
}

// New builds a limiter allowing up to max attempts per window.
func New(max int, window time.Duration) *Limiter {
	return &Limiter{
		max:    max,
		window: window,
		hits:   map[string]*entry{},
	}
}

// Max reports the number of attempts allowed per window. It is fixed at
// construction, so reads are safe without the lock.
func (l *Limiter) Max() int {
	return l.max
}

// Allow records an attempt and reports whether it fits inside the window.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	e, ok := l.hits[key]
	if !ok {
		e = &entry{}
		l.hits[key] = e
	}

	keep := e.times[:0]
	for _, t := range e.times {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	e.times = keep
	e.lastSeen = now
	l.prune(now)

	if len(keep) >= l.max {
		return false
	}
	e.times = append(keep, now)
	return true
}

// prune drops keys idle for at least one window, bounding memory. Call it
// whenever the map grows; the map stays small so a full sweep is cheap.
func (l *Limiter) prune(now time.Time) {
	for k, e := range l.hits {
		if now.Sub(e.lastSeen) > l.window {
			delete(l.hits, k)
		}
	}
}
