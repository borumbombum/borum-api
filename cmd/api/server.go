// Server helpers: printRoutes prints the startup banner listing every endpoint.
// Server lifecycle lives in main.go; routes are declared in routes.go.
package main

import "fmt"

func printRoutes(routes []apiRoute) {
	baseURL := "http://127.0.0.1:" + apiPort
	fmt.Println("Borum API:")
	for i, rt := range routes {
		path := routePath(rt.path)
		branch := "├─"
		if i == len(routes)-1 {
			branch = "└─"
		}
		fmt.Printf("%s %s %s  %s%s\n", branch, rt.method, path, baseURL, path)
	}
}
