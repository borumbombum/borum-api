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

const apiVersion = "v1"

type apiRoute struct {
	method  string
	path    string
	handler http.HandlerFunc
}

var apiRoutes = []apiRoute{
	{http.MethodGet, "/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"borum-api"}`))
	}},
	{http.MethodGet, "/cms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"endpoint":"cms"}`))
	}},
}

func main() {
	app := pocketbase.New()

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

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Any("/{path...}", func(e *core.RequestEvent) error {
			r.ServeHTTP(e.Response, e.Request)
			return nil
		})
		go printRoutes(se.Server.Addr)
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func printRoutes(addr string) {
	time.Sleep(50 * time.Millisecond)
	baseURL := "http://" + addr
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
