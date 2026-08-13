package auth

import (
	"sync"
	"time"
)

// loginLimiter is a small in-memory sliding-window limiter keyed per client
// (remote address + email). It exists only to slow brute-force attempts on the
// public login endpoint; a per-process counter is plenty for a single-user app.
type loginLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

const (
	loginMaxAttempts = 5
	loginWindow      = time.Minute
)

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{hits: map[string][]time.Time{}}
}

// allow records an attempt and reports whether it fits inside the window.
func (l *loginLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-loginWindow)
	window := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			window = append(window, t)
		}
	}
	if len(window) >= loginMaxAttempts {
		l.hits[key] = window
		return false
	}
	l.hits[key] = append(window, now)
	return true
}
