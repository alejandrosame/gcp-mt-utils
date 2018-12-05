package main

import (
    "net/http"

    "github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
    // Create the standar middleware to be used in our app
    standardMiddleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

    mux := http.NewServeMux()
    mux.HandleFunc("/", app.home)
    mux.HandleFunc("/pair", app.showPair)
    mux.HandleFunc("/pair/create", app.createPair)

    // TODO: Change template file location to use absolute path based on the current file location
    fileServer := http.FileServer(http.Dir("./ui/static/"))
    mux.Handle("/static/", http.StripPrefix("/static", fileServer))

    return standardMiddleware.Then(mux)
}