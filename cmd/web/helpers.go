package main

import (
    "bytes"
    "fmt"
    "net/http"
    "runtime/debug"
    "time"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    "github.com/justinas/nosurf"
)

func (app *application) serverError(w http.ResponseWriter, err error) {
    trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
    app.errorLog.Output(2, trace)

    http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}


func (app *application) clientError(w http.ResponseWriter, status int) {
    http.Error(w, http.StatusText(status), status)
}


func (app *application) clientErrorDetailed(w http.ResponseWriter, status int, errorDetail string) {
    explanation := fmt.Sprintf("%s: %s", http.StatusText(status), errorDetail)
    http.Error(w, explanation, status)
}


func (app *application) notFound(w http.ResponseWriter) {
    app.clientError(w, http.StatusNotFound)
}


func (app *application) addDefaultData(td *templateData, r *http.Request) *templateData {
    if td == nil {
        td = &templateData{}
    }

    td.AuthenticatedUser = app.authenticatedUser(r)
    td.Languages = app.selectedLanguages(r)
    td.CSRFToken = nosurf.Token(r)
    td.CurrentYear = time.Now().Year()
    // Recover feedback message for the user
    td.Flash = app.session.PopString(r, "flash")
    return td
}


func (app *application) render(w http.ResponseWriter, r *http.Request, name string, td *templateData) {
    ts, ok := app.templateCache[name]
    if !ok {
        app.serverError(w, fmt.Errorf("The template %s does not exist", name))
        return
    }

    buf := new(bytes.Buffer)

    // Write the template to the buffer, instead of straight to the
    // http.ResponseWriter. If there's an error, call our serverError helper and then
    // return.
    err := ts.Execute(buf, app.addDefaultData(td, r))
    if err != nil {
        app.serverError(w, err)
        return
    }

    buf.WriteTo(w)
}


func (app *application) authenticatedUser(r *http.Request) *models.User {
    user, ok := r.Context().Value(contextKeyUser).(*models.User)
    if !ok {
        return nil
    }
    return user
}


func (app *application) adminUser(r *http.Request) bool {
    user, ok := r.Context().Value(contextKeyUser).(*models.User)
    if !ok {
        return false
    }
    return user.Admin || user.Super
}


func (app *application) validatorUser(r *http.Request) bool {
    user, ok := r.Context().Value(contextKeyUser).(*models.User)
    if !ok {
        return false
    }
    return user.Validator || user.Admin || user.Super
}


func (app *application) translatorUser(r *http.Request) bool {
    user, ok := r.Context().Value(contextKeyUser).(*models.User)
    if !ok {
        return false
    }
    return user.Translator || user.Admin || user.Super
}


func (app *application) selectedLanguages(r *http.Request) string {
    languages, ok := r.Context().Value(contextKeyLanguages).(string)
    if !ok {
        return ""
    }
    return languages
}