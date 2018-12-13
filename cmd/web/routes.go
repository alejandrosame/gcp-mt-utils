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

    // Create middleware to take care of role access
    validatorMiddleware := dynamicMiddleware.Append(app.requireAuthenticatedUser, app.requireValidatorUser)
    translatorMiddleware := dynamicMiddleware.Append(app.requireAuthenticatedUser, app.requireTranslatorUser)
    adminMiddleware := dynamicMiddleware.Append(app.requireAuthenticatedUser, app.requireAdminUser)

    mux := pat.New()
    mux.Get("/", dynamicMiddleware.ThenFunc(app.home))

    // Validator routes
    mux.Get("/pair/create", validatorMiddleware.ThenFunc(app.createPairForm))
    mux.Post("/pair/create", validatorMiddleware.ThenFunc(app.createPair))
    mux.Get("/pair/validate/:id", validatorMiddleware.ThenFunc(app.validatePairForm))
    mux.Post("/pair/validate/:id", validatorMiddleware.ThenFunc(app.validatePair))
    mux.Get("/pair/validate", validatorMiddleware.ThenFunc(app.chooseLanguagesValidatePairForm))
    mux.Post("/pair/validate", validatorMiddleware.ThenFunc(app.chooseLanguagesValidatePair))
    mux.Get("/pair/:id", validatorMiddleware.ThenFunc(app.showPair))
    mux.Get("/pair", validatorMiddleware.ThenFunc(app.showPairs))
    mux.Get("/pairs/upload", validatorMiddleware.ThenFunc(app.uploadPairsForm))
    mux.Post("/pairs/upload", validatorMiddleware.ThenFunc(app.uploadPairs))

    // Translator routes
    mux.Get("/translate", translatorMiddleware.ThenFunc(app.translateForm))
    mux.Post("/translate", translatorMiddleware.ThenFunc(app.translateOrExport))

    // Admin routes
    mux.Get("/model", adminMiddleware.ThenFunc(app.showModels))
    mux.Get("/dataset/delete/:name", adminMiddleware.ThenFunc(app.deleteDatasetForm))
    mux.Post("/dataset/delete/:name", adminMiddleware.ThenFunc(app.deleteDataset))
    mux.Get("/dataset/train/:name", adminMiddleware.ThenFunc(app.trainDatasetForm))
    mux.Post("/dataset/train/:name", adminMiddleware.ThenFunc(app.trainDataset))
    mux.Get("/dataset", adminMiddleware.ThenFunc(app.showDatasets))
    mux.Get("/train/status", adminMiddleware.ThenFunc(app.showTrainingStatus))
    mux.Get("/user/signup/invitation/generate", adminMiddleware.ThenFunc(app.generateInvitationLinkForm))
    mux.Post("/user/signup/invitation/generate", adminMiddleware.ThenFunc(app.generateInvitationLink))

    
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