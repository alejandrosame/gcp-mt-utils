package main

import (
    "html/template"
    "path/filepath"
    "time"

    "github.com/alejandrosame/gcp-mt-utils/pkg/automl"
    "github.com/alejandrosame/gcp-mt-utils/pkg/forms"
    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

// Define a templateData type to act as the holding structure for
// any dynamic data that we want to pass to our HTML templates.
type templateData struct {
    AuthenticatedUser *models.User
    CSRFToken         string
    CurrentYear       int
    Flash             string
    Form              *forms.Form
    Pair              *models.Pair
    Pairs             []*models.Pair
    Models            []*automl.Model
    TrainReport       *automl.TrainOperationReport
    Datasets          []*automl.Dataset
}


func humanDate(t time.Time) string {
    return t.Format("02 Jan 2006 at 15:04")
}

// Initialize a template.FuncMap object and store it in a global variable. This is
// essentially a string-keyed map which acts as a lookup between the names of our
// custom template functions and the functions themselves.
var functions = template.FuncMap{
    "humanDate": humanDate,
}


// In memory template cache as a map
func newTemplateCache() (map[string]*template.Template, error) {
    cache := map[string]*template.Template{}

    // TODO: Change template file location to use absolute path based on the current file location
    pages, err := filepath.Glob("./ui/html/*.page.tmpl")
    if err != nil {
        return nil, err
    }

    for _, page := range pages {
        name := filepath.Base(page)

        // Register custom functions before parsing current page
        ts, err := template.New(name).Funcs(functions).ParseFiles(page)
        if err != nil {
            return nil, err
        }

        ts, err = ts.ParseGlob("./ui/html/*.layout.tmpl")
        if err != nil {
            return nil, err
        }

        ts, err = ts.ParseGlob("./ui/html/*.partial.tmpl")
        if err != nil {
            return nil, err
        }

        cache[name] = ts
    }

    return cache, nil
}