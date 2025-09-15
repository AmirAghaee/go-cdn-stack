package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Serve files from current directory
	fs := http.FileServer(http.Dir("./public"))

	http.Handle("/", fs)

	fmt.Println("Origin server running on :8081")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		panic(err)
	}
}
