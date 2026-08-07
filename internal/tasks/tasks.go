// Package tasks registers scheduled jobs on PocketBase's built-in cron
// scheduler (pb.Cron). Jobs are wired from the API server's sync.Once so
// they run exactly once per process.
package tasks

import (
	"log"

	"github.com/pocketbase/pocketbase"
)

// Register wires scheduled jobs onto the PocketBase cron scheduler.
// PB cron uses a standard 5-field expression (minute hour day month weekday)
// and ticks once per minute; jobs run only on the exact matching times.
// Only registers callbacks; it never touches the DB or blocks at call time.
// Jobs return nothing, so handle errors inside each job body.
func Register(pb *pocketbase.PocketBase) {
	// Every minute.
	pb.Cron().MustAdd("heartbeat", "* * * * *", func() {
		log.Println("borum-api: scheduler heartbeat")
	})

	// Top of every hour: example of a job touching the database through pb.
	pb.Cron().MustAdd("count-superusers", "0 * * * *", func() {
		n, err := pb.CountRecords("_superusers")
		if err != nil {
			log.Printf("borum-api: count-superusers error: %v", err)
			return
		}
		log.Printf("borum-api: superusers=%d", n)
	})
}
