package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Serve files from ./public
	fs := http.FileServer(http.Dir("./public"))

	// Wrap file server with logging middleware
	http.Handle("/", loggingMiddleware(fs))

	fmt.Println("Origin server running on :8081")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		panic(err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request:", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
