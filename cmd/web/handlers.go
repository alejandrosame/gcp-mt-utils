package main

import (
    "fmt"
    "net/http"
    //"regexp"
    "strconv"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        app.notFound(w)
        return
    }

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

    id, err := strconv.Atoi(r.URL.Query().Get("id"))
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

    app.render(w, r, "show.page.tmpl", &templateData{Pair: p})
}

func (app *application) createPair(w http.ResponseWriter, r *http.Request) {

    if r.Method != "POST" {
        w.Header().Set("Allow", "POST")
        app.clientError(w, http.StatusMethodNotAllowed)
        return
    }

    // Dummy data
    sourceLanguage := "EN"
    targetLanguage := "ES"
    sourceText := "This is good"
    targetText := "Esto es bueno"

    id, err := app.pairs.Insert(sourceLanguage, targetLanguage, sourceText, targetText)
    if err != nil {
        app.serverError(w, err)
        return
    }

    http.Redirect(w, r, fmt.Sprintf("/pair?id=%d", id), http.StatusSeeOther)
}