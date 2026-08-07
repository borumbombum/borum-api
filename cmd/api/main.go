// Command borum-api runs the PocketBase backend (default :8090) plus a custom
// chi HTTP API on :8091. The custom API is the only surface exposed to the
// public; PocketBase stays private and is used as the data layer.
package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// apiVersion prefixes all custom API routes except the root health check.
	apiVersion = "v1"
	// apiPort is the custom chi API port. PocketBase's own port (:8090) is
	// configured at runtime via the serve --http flag, not here.
	apiPort = "8091"
)

// app holds our dependencies and server state,
// removing the need for a global pocketbaseApp variable.
type app struct {
	pb    *pocketbase.PocketBase
	srv   *http.Server
	start sync.Once
}

// main wires PocketBase to the custom API: it builds the app dependency
// holder, then hooks the server startup and scheduled tasks to OnServe
// (fired after the DB is initialized) and graceful shutdown to OnTerminate.
func main() {
	pb := pocketbase.New()

	api := &app{pb: pb}

	// Triggered after DB initializes, safe to start custom API.
	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		api.startAPIServer()
		return se.Next()
	})

	// Graceful shutdown with a 5-second context timeout.
	pb.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
		if api.srv != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := api.srv.Shutdown(ctx); err != nil {
				log.Printf("Borum API shutdown error: %v", err)
			}
		}
		return te.Next()
	})

	if err := pb.Start(); err != nil {
		log.Fatal(err)
	}
}
