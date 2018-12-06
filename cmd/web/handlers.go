package main

import (
    "fmt"
    "net/http"
    //"regexp"
    "strconv"

    "github.com/alejandrosame/gcp-mt-utils/pkg/forms"
    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
    p, err := app.pairs.Latest()
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "home.page.tmpl", &templateData{Pairs: p})
}


func (app *application) showPair(w http.ResponseWriter, r *http.Request) {

    /* For future reference of text validation
    id := r.URL.Query().Get("id")

    if m, _ := regexp.MatchString("^[a-zA-Z1-9\\-]+$", id); !m {
        app.notFound(w)
        return
    }
    */

    id, err := strconv.Atoi(r.URL.Query().Get(":id"))
    if err != nil || id < 1 {
        app.notFound(w)
        return
    }

    p, err := app.pairs.Get(id)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "show.page.tmpl", &templateData{
        Pair:  p,
    })
}


func (app *application) createPairForm(w http.ResponseWriter, r *http.Request) {
    app.render(w, r, "create.page.tmpl", &templateData{Form: forms.New(nil)})
}


func (app *application) createPair(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("sourceLanguage", "targetLanguage", "sourceText", "targetText")
    // Max number of chars for text input
    maxChar := 10000
    form.MaxLength("sourceText", maxChar)
    form.MaxLength("targetText", maxChar)
    // Languages codes to check
    form.PermittedValues("sourceLanguage", "EN", "ES", "FR", "PT", "SW")
    form.PermittedValues("targetLanguage", "EN", "ES", "FR", "PT", "SW")

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.render(w, r, "create.page.tmpl", &templateData{Form: form})
        return
    }

    sourceLanguage := form.Get("sourceLanguage")
    targetLanguage := form.Get("targetLanguage")
    sourceText := form.Get("sourceText")
    targetText := form.Get("targetText")

    id, err := app.pairs.Insert(sourceLanguage, targetLanguage, sourceText, targetText)
    if err != nil {
        app.serverError(w, err)
        return
    }

    // Add feedback for the user as session information
    app.session.Put(r, "flash", "Pair successfully created!")

    http.Redirect(w, r, fmt.Sprintf("/pair/%d", id), http.StatusSeeOther)
}