// Server helpers: printRoutes prints the startup banner listing every endpoint.
// Server lifecycle lives in main.go; routes are declared in routes.go.
package main

import "fmt"

func printRoutes(routes []apiRoute, addr string) {
	fmt.Printf("Borum API listening on %s\n", addr)
	for i, rt := range routes {
		path := rt.path
		branch := "├─"
		if i == len(routes)-1 {
			branch = "└─"
		}
		fmt.Printf("%s %s %s\n", branch, rt.method, path)
	}
}
