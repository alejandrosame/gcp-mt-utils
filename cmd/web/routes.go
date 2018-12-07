package main

import (
    "net/http"

    "github.com/bmizerany/pat"
    "github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
    // Create the standar middleware to be used in our app
    standardMiddleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

    // Create middleware to be applied only on the dynamic application routes.
    dynamicMiddleware := alice.New(app.session.Enable, noSurf, app.authenticate)

    mux := pat.New()
    mux.Get("/", dynamicMiddleware.ThenFunc(app.home))
    mux.Get("/pair/create", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.createPairForm))
    mux.Post("/pair/create", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.createPair))
    mux.Get("/pair/:id", dynamicMiddleware.ThenFunc(app.showPair))
    mux.Get("/pairs/upload", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.uploadPairsForm))
    mux.Post("/pairs/upload", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.uploadPairs))

    // User session routes
    mux.Get("/user/signup", dynamicMiddleware.ThenFunc(app.signupUserForm))
    mux.Post("/user/signup", dynamicMiddleware.ThenFunc(app.signupUser))
    mux.Get("/user/login", dynamicMiddleware.ThenFunc(app.loginUserForm))
    mux.Post("/user/login", dynamicMiddleware.ThenFunc(app.loginUser))
    mux.Post("/user/logout", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.logoutUser))

    // TODO: Change template file location to use absolute path based on the current file location
    fileServer := http.FileServer(http.Dir("./ui/static/"))
    mux.Get("/static/", http.StripPrefix("/static", fileServer))

    return standardMiddleware.Then(mux)
}