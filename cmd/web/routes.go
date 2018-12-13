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
    mux.Get("/pair/validate/:id", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.validatePairForm))
    mux.Post("/pair/validate/:id", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.validatePair))
    mux.Get("/pair/validate", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.chooseLanguagesValidatePairForm))
    mux.Post("/pair/validate", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.chooseLanguagesValidatePair))
    mux.Get("/pair/:id", dynamicMiddleware.ThenFunc(app.showPair))
    mux.Get("/pairs/upload", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.uploadPairsForm))
    mux.Post("/pairs/upload", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.uploadPairs))
    mux.Get("/translate", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.translateForm))
    mux.Post("/translate", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.translateOrExport))
    mux.Get("/model", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.showModels))
    mux.Get("/dataset/delete/:name", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.deleteDatasetForm))
    mux.Post("/dataset/delete/:name", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.deleteDataset))
    mux.Get("/dataset/train/:name", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.trainDatasetForm))
    mux.Post("/dataset/train/:name", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.trainDataset))
    mux.Get("/dataset", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.showDatasets))
    mux.Get("/train/status", dynamicMiddleware.Append(app.requireAuthenticatedUser).ThenFunc(app.showTrainingStatus))
    
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