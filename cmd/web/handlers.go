package main

import (
    "fmt"
    "net/http"
    "regexp"
)

func home(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }

    w.Write([]byte("Hello from GCP MT Utils\n"))
}

// Add a showSnippet handler function.
func showPairs(w http.ResponseWriter, r *http.Request) {

    id := r.URL.Query().Get("id")

    if m, _ := regexp.MatchString("^[a-zA-Z1-9\\-]+$", id); !m {
        http.NotFound(w, r)
        return
    }

    fmt.Fprintf(w, "Display a specific training pair file with ID %s...\n", id)
}

// Add a createSnippet handler function.
func loadPairs(w http.ResponseWriter, r *http.Request) {

    if r.Method != "POST" {
        w.Header().Set("Allow", "POST")
        http.Error(w, "Method Not Allowed", 405)
        return
    }

    w.Write([]byte("Load a new training pair file...\n"))
}