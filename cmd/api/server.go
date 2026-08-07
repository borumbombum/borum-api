// Server lifecycle: startAPIServer spins up the HTTP listener (guarded by
// sync.Once, so repeated OnServe calls can't double-start) and registers
// scheduled tasks; printRoutes prints the startup banner listing every endpoint.
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/borumbombum/borum-api/internal/tasks"
)

// startAPIServer spins up the HTTP server and scheduled tasks. It is guarded by
// sync.Once because OnServe can fire multiple times (e.g., HTTP + HTTPS).
func (a *app) startAPIServer() {
	a.start.Do(func() {
		// Register cron jobs before binding the port; the order is arbitrary
		// (cron and the HTTP listener are independent) but runs exactly once.
		tasks.Register(a.pb)

		routes := a.apiRoutes()

		a.srv = &http.Server{
			Addr:         "127.0.0.1:" + apiPort,
			Handler:      a.routes(),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		}

		go func() {
			if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("API server error: %v", err)
			}
		}()

		// Keep the 50ms delay in a goroutine so our API banner
		// prints cleanly *after* the PocketBase startup banner.
		go a.printRoutes(routes)
	})
}

func (a *app) printRoutes(routes []apiRoute) {
	time.Sleep(50 * time.Millisecond)
	baseURL := "http://127.0.0.1:" + apiPort
	fmt.Println("Borum API:")
	for i, rt := range routes {
		path := rt.path
		if path != "/" {
			path = "/" + apiVersion + path
		}
		branch := "├─"
		if i == len(routes)-1 {
			branch = "└─"
		}
		fmt.Printf("%s %s %s  %s%s\n", branch, rt.method, path, baseURL, path)
	}
}
