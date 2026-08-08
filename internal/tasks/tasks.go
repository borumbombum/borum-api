// Package tasks provides a minimal in-process scheduler for background jobs.
// It is intentionally dependency-free: jobs are registered ahead of time and
// run on a fixed tick. Once a database layer (Turso) is wired in, jobs that
// need it can hold a reference to the DB client.
package tasks

import (
	"log"
	"sync"
	"time"
)

// tickInterval controls how often pending jobs are considered for execution.
const tickInterval = time.Minute

// job is a single scheduled task: a name plus a run function.
type job struct {
	name  string
	every time.Duration
	fn    func()
	last  time.Time
}

var (
	mu   sync.Mutex
	jobs []*job
)

// Add registers a job that runs every time.Duration, starting from the time it
// is added. It is not safe to call Add after Register has started.
func Add(name string, every time.Duration, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	jobs = append(jobs, &job{name: name, every: every, last: time.Now(), fn: fn})
}

// Register starts the scheduler loop in its own goroutine. It fires each job
// whose next due tick has passed and returns immediately. Jobs run
// sequentially, so a long job delays later ones.
//
// It returns the number of registered jobs (including the auto-registered
// heartbeat), so callers can tell when the scheduler has nothing to run.
func Register() int {
	// Guard against double-starting the loop.
	mu.Lock()
	if started {
		n := len(jobs)
		mu.Unlock()
		return n
	}
	started = true
	mu.Unlock()

	Add("heartbeat", tickInterval, func() {
		log.Println("borum-api: scheduler heartbeat")
	})

	go loop()

	mu.Lock()
	defer mu.Unlock()
	return len(jobs)
}

var started bool

func loop() {
	heartbeat := time.NewTicker(tickInterval)
	defer heartbeat.Stop()

	for range heartbeat.C {
		runDue()
	}
}

func runDue() {
	now := time.Now()
	var due []job

	mu.Lock()
	for _, j := range jobs {
		if now.Sub(j.last) >= j.every {
			j.last = now
			due = append(due, *j)
		}
	}
	mu.Unlock()

	for _, j := range due {
		runSafe(j)
	}
}

func runSafe(j job) {
	// A panicking job must not take down the whole scheduler loop.
	defer func() {
		if err := recover(); err != nil {
			log.Printf("borum-api: scheduler job %q panicked: %v", j.name, err)
		}
	}()
	j.fn()
}
