package main

import (
    "fmt"
    "net/http"
    "strconv"

    "github.com/alejandrosame/gcp-mt-utils/pkg/files"
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

func (app *application) signupUserForm(w http.ResponseWriter, r *http.Request) {
    app.render(w, r, "signup.page.tmpl", &templateData{
        Form: forms.New(nil),
    })
}

func (app *application) signupUser(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("name", "email", "password")
    form.MatchesPattern("email", forms.EmailRX)
    form.MinLength("password", 10)

    // If there are any errors, redisplay the signup form.
    if !form.Valid() {
        app.render(w, r, "signup.page.tmpl", &templateData{Form: form})
        return
    }

    // Try to create a new user record in the database. If the email already exists
    // add an error message to the form and re-display it.
    err = app.users.Insert(form.Get("name"), form.Get("email"), form.Get("password"))
    if err == models.ErrDuplicateEmail {
        form.Errors.Add("email", "Address is already in use")
        app.render(w, r, "signup.page.tmpl", &templateData{Form: form})
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    // Otherwise add a confirmation flash message to the session confirming that
    // their signup worked and asking them to log in.
    app.session.Put(r, "flash", "Your signup was successful. Please log in.")

    // And redirect the user to the login page.
    http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) loginUserForm(w http.ResponseWriter, r *http.Request) {
    app.render(w, r, "login.page.tmpl", &templateData{
        Form: forms.New(nil),
    })
}

func (app *application) loginUser(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    // Check whether the credentials are valid. If they're not, add a generic error
    // message to the form failures map and re-display the login page.
    form := forms.New(r.PostForm)
    id, err := app.users.Authenticate(form.Get("email"), form.Get("password"))
    if err == models.ErrInvalidCredentials {
        form.Errors.Add("generic", "Email or Password is incorrect")
        app.render(w, r, "login.page.tmpl", &templateData{Form: form})
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    // Add the ID of the current user to the session, so that they are now 'logged in'.
    app.session.Put(r, "userID", id)

    // Redirect the user to the create pair page.
    http.Redirect(w, r, "/pair/create", http.StatusSeeOther)
}

func (app *application) logoutUser(w http.ResponseWriter, r *http.Request) {
    // Remove the userID from the session data so that the user is 'logged out'.
    app.session.Remove(r, "userID")
    // Add a flash message to the session to confirm to the user that they've been logged out.
    app.session.Put(r, "flash", "You've been logged out successfully!")
    http.Redirect(w, r, "/", 303)
}


func (app *application) uploadPairsForm(w http.ResponseWriter, r *http.Request) {
    app.render(w, r, "upload.page.tmpl", &templateData{Form: forms.New(nil)})
}


func (app *application) uploadPairs(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("sourceLanguage", "targetLanguage")
    // Languages codes to check
    form.PermittedValues("sourceLanguage", "EN", "ES", "FR", "PT", "SW")
    form.PermittedValues("targetLanguage", "EN", "ES", "FR", "PT", "SW")

    sourceLanguage := form.Get("sourceLanguage")
    targetLanguage := form.Get("targetLanguage")
    tmp_file, fileType := form.ProcessFileUpload(w, r, *app.maxUploadSize, *app.uploadPath, app.infoLog, app.errorLog)

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.render(w, r, "upload.page.tmpl", &templateData{Form: form})
        return
    }

    var tpfile *files.TranslationPairFile = nil

    if fileType == "TSV" {
        tpfile = files.ReadPairsFromTsv(tmp_file, sourceLanguage, targetLanguage)
    }
    if fileType == "XLSX" {
        tpfile = files.ReadPairsFromXlsx(tmp_file, sourceLanguage, targetLanguage)
    }

    if !tpfile.Valid() {
        form.Errors["fileName"] = tpfile.Errors["fileName"]
        app.render(w, r, "upload.page.tmpl", &templateData{Form: form})
        return
    }

    count, err := app.pairs.BulkInsert(tpfile.Pairs)
    if err != nil {
        app.serverError(w, err)
        return
    }

    // Add feedback for the user as session information
    app.session.Put(r, "flash", fmt.Sprintf("%d Pairs successfully uploaded!", count))

    http.Redirect(w, r, "/", http.StatusSeeOther)
}