package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	apiVersion = "v1"
	apiPort    = "8091"
)

var pocketbaseApp *pocketbase.PocketBase

type apiRoute struct {
	method  string
	path    string
	handler http.HandlerFunc
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := pocketbaseApp.CountRecords("_superusers")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"degraded","service":"borum-api","database":"unreachable"}`))
		return
	}
	w.Write([]byte(`{"status":"ok","service":"borum-api","database":"ok"}`))
}

var apiRoutes = []apiRoute{
	{http.MethodGet, "/", healthHandler},
	{http.MethodGet, "/cms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"endpoint":"cms"}`))
	}},
}

func main() {
	pocketbaseApp = pocketbase.New()

	go startAPIServer()

	pocketbaseApp.OnServe().BindFunc(func(se *core.ServeEvent) error {
		go printRoutes()
		return se.Next()
	})

	if err := pocketbaseApp.Start(); err != nil {
		log.Fatal(err)
	}
}

func startAPIServer() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	for _, rt := range apiRoutes {
		path := rt.path
		if path != "/" {
			path = "/" + apiVersion + path
		}
		r.Method(rt.method, path, rt.handler)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"status":"not_found"}`))
	})

	if err := http.ListenAndServe("127.0.0.1:"+apiPort, r); err != nil {
		log.Fatal(err)
	}
}

func printRoutes() {
	time.Sleep(50 * time.Millisecond)
	baseURL := "http://127.0.0.1:" + apiPort
	fmt.Println("Borum API:")
	for i, rt := range apiRoutes {
		path := rt.path
		if path != "/" {
			path = "/" + apiVersion + path
		}
		branch := "├─"
		if i == len(apiRoutes)-1 {
			branch = "└─"
		}
		fmt.Printf("%s %s %s  %s%s\n", branch, rt.method, path, baseURL, path)
	}
}
