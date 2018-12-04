package main

import "net/http"

func (app *application) routes() *http.ServeMux {
    mux := http.NewServeMux()
    mux.HandleFunc("/", app.home)
    mux.HandleFunc("/pairs", app.showPairs)
    mux.HandleFunc("/pairs/load", app.loadPairs)

    // TODO: Change template file location to use absolute path based on the current file location
    fileServer := http.FileServer(http.Dir("./ui/static/"))
    mux.Handle("/static/", http.StripPrefix("/static", fileServer))

    return mux
}