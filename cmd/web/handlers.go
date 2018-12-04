package main

import (
    "fmt"
    //"html/template"
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

    s, err := app.pairs.Latest()
    if err != nil {
        app.serverError(w, err)
        return
    }

    for _, pair := range s {
        fmt.Fprintf(w, "%v\n", pair)
    }

    /*
    if r.URL.Path != "/" {
        app.notFound(w)
        return
    }

    // TODO: Change template file location to use absolute path based on the current file location
    files := []string{
        "./ui/html/home.page.tmpl",
        "./ui/html/base.layout.tmpl",
        "./ui/html/footer.partial.tmpl",
    }

    ts, err := template.ParseFiles(files...)
    if err != nil {
        app.errorLog.Println(err.Error())
        app.serverError(w, err)
        return
    }

    err = ts.Execute(w, nil)
    if err != nil {
        app.errorLog.Println(err.Error())
        app.serverError(w, err)
    }
    */
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

    s, err := app.pairs.Get(id)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    fmt.Fprintf(w, "%v", s)
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