package main

import (
    "net/http"

    "github.com/bmizerany/pat"
    "github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
    // Create the standar middleware to be used in our app
    standardMiddleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

    mux := pat.New()
    mux.Get("/", http.HandlerFunc(app.home))
    mux.Get("/pair/create", http.HandlerFunc(app.createPairForm))
    mux.Post("/pair/create", http.HandlerFunc(app.createPair))
    mux.Get("/pair/:id", http.HandlerFunc(app.showPair))

    // TODO: Change template file location to use absolute path based on the current file location
    fileServer := http.FileServer(http.Dir("./ui/static/"))
    mux.Get("/static/", http.StripPrefix("/static", fileServer))

    return standardMiddleware.Then(mux)
}