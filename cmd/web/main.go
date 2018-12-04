package main

import (
    "log"
    "net/http"
)

func main() {
    // Register the two new handler functions and corresponding URL patterns with
    // the servemux, in exactly the same way that we did before.
    mux := http.NewServeMux()
    mux.HandleFunc("/", home)
    mux.HandleFunc("/pairs", showPairs)
    mux.HandleFunc("/pairs/load", loadPairs)

    log.Println("Starting server on :4000")
    err := http.ListenAndServe(":4000", mux)
    log.Fatal(err)
}