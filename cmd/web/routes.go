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
    dynamicMiddleware := alice.New(app.session.Enable, noSurf, app.authenticate, app.selectLanguages)

    // Create middleware to take care of role access
    validatorMiddleware := dynamicMiddleware.Append(app.requireAuthenticatedUser, app.requireValidatorUser,
                                                    app.requireSelectedLanguages)
    translatorMiddleware := dynamicMiddleware.Append(app.requireAuthenticatedUser, app.requireTranslatorUser,
                                                     app.requireSelectedLanguages)
    adminMiddleware := dynamicMiddleware.Append(app.requireAuthenticatedUser, app.requireAdminUser,
                                                app.requireSelectedLanguages)

    mux := pat.New()
    mux.Get("/", dynamicMiddleware.ThenFunc(app.home))

    // Selection routes
    mux.Get("/language/:code", dynamicMiddleware.ThenFunc(app.setLanguage))

    // Validator routes
    mux.Get("/pair/create", validatorMiddleware.ThenFunc(app.createPairForm))
    mux.Post("/pair/create", validatorMiddleware.ThenFunc(app.createPair))
    mux.Get("/pair/validate/:id", validatorMiddleware.ThenFunc(app.validatePairForm))
    mux.Post("/pair/validate/:id", validatorMiddleware.ThenFunc(app.validatePair))
    mux.Get("/pair/validate", validatorMiddleware.ThenFunc(app.initValidatePair))
    mux.Get("/pair/:id", validatorMiddleware.ThenFunc(app.showPair))
    mux.Get("/pair", validatorMiddleware.ThenFunc(app.showPairs))
    mux.Get("/pairs/upload", validatorMiddleware.ThenFunc(app.uploadPairsForm))
    mux.Post("/pairs/upload", validatorMiddleware.ThenFunc(app.uploadPairs))
    mux.Get("/pairs/export", validatorMiddleware.ThenFunc(app.exportValidatedPairsForm))
    mux.Post("/pairs/export", validatorMiddleware.ThenFunc(app.exportValidatedPairs))

    // Translator routes
    mux.Get("/translate/export", translatorMiddleware.ThenFunc(app.exportTranslation))
    mux.Get("/translate/:source", translatorMiddleware.ThenFunc(app.translate))
    mux.Get("/translate", translatorMiddleware.ThenFunc(app.translatePage))

    // Admin routes
    mux.Get("/model/delete/:name", adminMiddleware.ThenFunc(app.deleteModelForm))
    mux.Post("/model/delete/:name", adminMiddleware.ThenFunc(app.deleteModel))
    mux.Get("/model", adminMiddleware.ThenFunc(app.showModels))
    mux.Get("/dataset/delete/:name", adminMiddleware.ThenFunc(app.deleteDatasetForm))
    mux.Post("/dataset/delete/:name", adminMiddleware.ThenFunc(app.deleteDataset))
    mux.Get("/dataset/train/:name", adminMiddleware.ThenFunc(app.trainDatasetForm))
    mux.Post("/dataset/train/:name", adminMiddleware.ThenFunc(app.trainDataset))
    mux.Get("/dataset", adminMiddleware.ThenFunc(app.showDatasets))
    mux.Get("/train/cancel/:name", adminMiddleware.ThenFunc(app.cancelTrainingOperationForm))
    mux.Post("/train/cancel/:name", adminMiddleware.ThenFunc(app.cancelTrainingOperation))
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