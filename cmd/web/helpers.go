package main

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

// Write an error message with traces
func (a *app) errorTrace(w http.ResponseWriter, err error) {
	trace := fmt.Sprint("%s\n%s", err.Error(), debug.Stack())
	a.errorLogger.Output(2, trace)
	// a.errorLogger.Printf("[ERROR_TRACE] %s", trace)
}
