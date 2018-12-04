package main

import (
    "fmt"
    "html/template"
    "net/http"
    "regexp"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
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
}

// Add a showSnippet handler function.
func (app *application) showPairs(w http.ResponseWriter, r *http.Request) {

    id := r.URL.Query().Get("id")

    if m, _ := regexp.MatchString("^[a-zA-Z1-9\\-]+$", id); !m {
        app.notFound(w)
        return
    }

    fmt.Fprintf(w, "Display a specific training pair file with ID %s...\n", id)
}

// Add a createSnippet handler function.
func (app *application) loadPairs(w http.ResponseWriter, r *http.Request) {

    if r.Method != "POST" {
        w.Header().Set("Allow", "POST")
        app.clientError(w, http.StatusMethodNotAllowed)
        return
    }

    w.Write([]byte("Load a new training pair file...\n"))
}