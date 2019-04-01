package main

import (
    "fmt"
    "encoding/json"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "time"

    "os"
    "bufio"

    "github.com/alejandrosame/gcp-mt-utils/pkg/automl"
    "github.com/alejandrosame/gcp-mt-utils/pkg/files"
    "github.com/alejandrosame/gcp-mt-utils/pkg/forms"
    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
    "github.com/alejandrosame/gcp-mt-utils/pkg/passwords"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
    app.render(w, r, "landing.page.tmpl", &templateData{})
}

func (app *application) aboutPage(w http.ResponseWriter, r *http.Request) {
    app.render(w, r, "about.page.tmpl", &templateData{})
}


func (app *application) showBooks(w http.ResponseWriter, r *http.Request) {
    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    b, err := app.pairs.GetBibleBooks(sourceLanguage, targetLanguage)
    if err != nil {
        app.serverError(w, err)
        return
    }

    stats, err := app.pairs.ValidationStatistics(sourceLanguage, targetLanguage)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "show.book.page.tmpl", &templateData{Books: b, ValidationStats: stats})
}


func (app *application) showPairs(w http.ResponseWriter, r *http.Request) {
    bookId, err := strconv.Atoi(r.URL.Query().Get(":bookid"))
    if err != nil || bookId < 1 {
        app.notFound(w)
        return
    }

    b, err := app.pairs.GetBook(bookId)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }


    chapterId, err := strconv.Atoi(r.URL.Query().Get(":chapterid"))
    if err != nil || chapterId < 1 || chapterId > b.Chapter {
        app.notFound(w)
        return
    }

    // Get next chapter info before rendering the page
    c := &models.BibleBook{}
    if chapterId < b.Chapter {
        c.ID = b.ID
        c.Chapter = chapterId + 1
    } else {
        _, err = app.pairs.GetBook(bookId+1)
        if err == models.ErrNoRecord {
            c = nil
        } else if err == nil {
            c.ID = bookId + 1
            c.Chapter = 1
        } else {
            app.serverError(w, err)
            return
        }
    }

    b.Chapter = chapterId

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    p, err := app.pairs.GetPairs(app.infoLog, sourceLanguage, targetLanguage, bookId, chapterId)
    if err != nil {
        app.serverError(w, err)
        return
    }

    var stats *models.ValidationStats = nil
    if (len(p) > 0){
        stats, err = app.pairs.ValidationStatisticsBookChapter(sourceLanguage, targetLanguage, p[0].Detail)
        if err != nil {
            app.serverError(w, err)
            return
        }
    }

    app.render(w, r, "show.pair.page.tmpl", &templateData{Pairs: p, ValidationStats: stats, Book: b, NextChapter: c})
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

    b, err := app.pairs.GetBookFromDetail(p.Detail)
    if err != nil {
        app.serverError(w, err)
        return
    }

    http.Redirect(w, r, fmt.Sprintf("/pair/book/%d/chapter/%d#%d", b.ID, b.Chapter, id), http.StatusSeeOther)
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
    form.Required("sourceText", "targetText", "sourceVersion", "targetVersion",
                  "detail")
    // Max number of chars for text input
    maxChar := 255
    form.MaxLength("sourceVersion", maxChar)
    form.MaxLength("targetVersion", maxChar)
    form.MaxLength("detail", maxChar)

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.render(w, r, "create.page.tmpl", &templateData{Form: form})
        return
    }

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")
    sourceText := form.Get("sourceText")
    targetText := form.Get("targetText")
    sourceVersion := form.Get("sourceVersion")
    targetVersion := form.Get("targetVersion")
    detail := form.Get("detail")

    id, err := app.pairs.Insert(sourceLanguage, sourceVersion, targetLanguage, targetVersion, detail,
                                sourceText, targetText)
    if err != nil {
        app.serverError(w, err)
        return
    }

    // Add feedback for the user as session information
    app.session.Put(r, "flash", "Pair successfully created!")

    http.Redirect(w, r, fmt.Sprintf("/pair/%d", id), http.StatusSeeOther)
}

func (app *application) editPairForm(w http.ResponseWriter, r *http.Request) {
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

    redirectPage := r.URL.Query().Get("redirect")
    if redirectPage == "validate" {
        redirectPage = fmt.Sprintf("/pair/validate/%d", id)
    } else {
        redirectPage = fmt.Sprintf("/pair/%d", id)
    }

    form := forms.New(url.Values{})
    form.Add("id", fmt.Sprintf("%d", p.ID))
    form.Add("sourceLanguage", p.SourceLanguage)
    form.Add("targetLanguage", p.TargetLanguage)
    form.Add("sourceText", p.SourceText)
    form.Add("targetText", p.TargetText)
    form.Add("sourceVersion", p.SourceVersion)
    form.Add("targetVersion", p.TargetVersion)
    form.Add("detail", p.Detail)
    if p.Comments.Valid{
        form.Add("comments", p.Comments.String)
    }else{
        form.Add("comments", "")
    }
    form.Add("updated", humanDate(p.Updated))
    form.Add("created", humanDate(p.Created))
    form.Add("redirectPage", redirectPage)

    app.render(w, r, "edit.page.tmpl", &templateData{Form: form})
}

func (app *application) editPair(w http.ResponseWriter, r *http.Request) {
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

    err = r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("sourceText", "targetText", "sourceVersion", "targetVersion", "detail", "redirectPage")

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.render(w, r, "edit.page.tmpl", &templateData{Form: form})
        return
    }

    sourceText := strings.Trim(form.Get("sourceText"), " \n")
    targetText := strings.Trim(form.Get("targetText"), " \n")
    comments := strings.Trim(form.Get("comments"), " \n")
    redirect := form.Get("redirectPage")

    savedComments := ""
    if p.Comments.Valid{
        savedComments = p.Comments.String
    }

    if sourceText != p.SourceText || targetText != p.TargetText {
        _, err = app.pairs.Edit(id, sourceText, targetText, comments)
        if err != nil {
            app.serverError(w, err)
            return
        }

        // Add feedback for the user as session information
        app.session.Put(r, "flash", "Pair successfully edited!")
    } else if savedComments != comments {
        _, err = app.pairs.EditComments(id, comments)
        if err != nil {
            app.serverError(w, err)
            return
        }

        app.session.Put(r, "flash", "Pair comments successfully edited!")
    } else {
        app.session.Put(r, "flash", "Pair content was not changed!")
    }

    http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (app *application) userCharacterLimitForm(w http.ResponseWriter, r *http.Request) {
    allLimit, err := app.users.GetRoleLimit("all")
    if err != nil {
        app.serverError(w, err)
        return
    }

    allUserLimits, err := app.users.GetAllUserLimits()
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "show.user.limit.page.tmpl",
               &templateData{
                    Form: nil,
                    RoleLimit: allLimit,
                    AllUserLimits: allUserLimits,
                })
}

func (app *application) updateGroupCharacterLimit(w http.ResponseWriter, r *http.Request) {
    group := r.URL.Query().Get(":group")
    if group == "" || (group != "all" && group != "validator" && group != "translator" && group != "admin") {
        app.notFound(w)
        return
    }

    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("limit")
    limitIntValue := form.MinIntValue("limit", 0)

    if !form.Valid() {
        app.session.Put(r, "flash",
                        "Limit value was not changed. Check that it is a valid number over 0 and not empty!")
        http.Redirect(w, r, "/user/limit", http.StatusSeeOther)
        return
    }

    _, err = app.users.UpdateRoleLimit(group, limitIntValue)
    if err != nil {
        app.serverError(w, err)
        return
    }

    if group == "all" {
        app.session.Put(r, "flash", "Base translation limit updated for all users!")
    } else{
        app.session.Put(r, "flash", fmt.Sprintf("Translation limit updated for %ss!", group))
    }
    http.Redirect(w, r, "/user/limit", http.StatusSeeOther)
}

func (app *application) updateUserCharacterLimit(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.URL.Query().Get(":id"))
    if err != nil || id < 1 {
        app.notFound(w)
        return
    }

    u, err := app.users.Get(id)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    err = r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("limit")
    limitIntValue := form.MinIntValue("limit", 0)

    if !form.Valid() {
        app.session.Put(r, "flash",
                        "Limit value was not changed. Check that it is a valid number over 0 and not empty!")
        http.Redirect(w, r, "/user/limit", http.StatusSeeOther)
        return
    }

    app.infoLog.Println(form.Get("limit"))
    app.infoLog.Println(limitIntValue)

    _, err = app.users.UpdateUserLimit(id, limitIntValue)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", fmt.Sprintf("Updated translation limit for user %s (%s)!", u.Name, u.Email))
    http.Redirect(w, r, "/user/limit", http.StatusSeeOther)
}

func (app *application) signupUserForm(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    if token == "" {
        app.notFound(w)
        return
    }

    found, err := app.invitations.TokenExists(token)
    if err == models.ErrTokenNotFound {
        app.session.Put(r, "flash", "Token expired or does not match invite.")
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    if found == false {
        app.notFound(w)
        return
    }

    f := forms.New(url.Values{})
    f.Add("inviteToken", token)

    app.render(w, r, "signup.page.tmpl", &templateData{
        Form: f,
    })
}

func (app *application) signupUser(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("name", "email", "password", "inviteToken")
    form.MatchesPattern("email", forms.EmailRX)
    form.MinLength("password", 10)

    // If there are any errors, redisplay the signup form.
    if !form.Valid() {
        app.render(w, r, "signup.page.tmpl", &templateData{Form: form})
        return
    }

    inv, err := app.invitations.CheckToken(form.Get("email"), form.Get("inviteToken"))
    if err == models.ErrTokenNotFound {
        app.session.Put(r, "flash", "Token expired or does not match invite.")
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    // Try to create a new user record in the database. If the email already exists
    // add an error message to the form and re-display it.
    err = app.users.Insert(form.Get("name"), form.Get("email"), form.Get("password"), inv.Role)
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

func (app *application) generateInvitationLinkForm(w http.ResponseWriter, r *http.Request) {

    app.render(w, r, "generate.signup.invitation.page.tmpl", &templateData{
        Form: forms.New(nil),
    })
}

func (app *application) generateInvitationLink(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("email", "role")
    form.PermittedValues("role", "validator", "translator", "admin")
    form.MatchesPattern("email", forms.EmailRX)

    if !form.Valid() {
        app.render(w, r, "generate.signup.invitation.page.tmpl", &templateData{Form: form})
        return
    }

    i, err := app.invitations.Insert(form.Get("email"), form.Get("role"))
    if err == models.ErrDuplicateEmail {
        form.Errors.Add("email", "Address is already in use")
        app.render(w, r, "generate.signup.invitation.page.tmpl", &templateData{Form: form})
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", "Invitation creation was successful.")

    app.render(w, r, "show.signup.invitation.page.tmpl", &templateData{ SignUpInvitation: i})
}

func (app *application) requestPasswordChangeForm(w http.ResponseWriter, r *http.Request) {
    app.render(w, r, "request.password.change.page.tmpl", &templateData{
        Form: forms.New(nil),
    })
}

func (app *application) requestPasswordChange(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("email")
    form.MatchesPattern("email", forms.EmailRX)

    if !form.Valid() {
        app.render(w, r, "request.password.change.page.tmpl", &templateData{Form: form})
        return
    }

    passwordChangeRequest, err := app.users.InsertPasswordChangeRequest(form.Get("email"))
    if err == models.ErrEmailNotFound {
        form.Errors.Add("email", "Address is not correct. Try again.")
        app.render(w, r, "request.password.change.page.tmpl", &templateData{Form: form})
        return
    } else if err == models.ErrTokenStillValid {
        app.session.Put(r, "flash", "It appears that a previous token is still active. Please, wait 10 minutes before requesting a new password change.")
        http.Redirect(w, r, "/user/password/request", http.StatusSeeOther)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    userName, err := app.users.GetUserName(form.Get("email"))
    if err != nil {
        app.serverError(w, err)
        return
    }

    passwords.SendEmail(app.infoLog, app.errorLog, app.reports, userName, passwordChangeRequest)

    app.session.Put(r, "flash", "Check your email for a password update link.")
    http.Redirect(w, r, "/user/password/request", http.StatusSeeOther)
}

func (app *application) passwordChangeForm(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    if token == "" {
        app.notFound(w)
        return
    }

    found, err := app.users.PasswordChangeTokenExists(token)
    if err == models.ErrTokenNotFound {
        app.session.Put(r, "flash", "Token expired or did not match user email.")
        http.Redirect(w, r, "/user/password/request", http.StatusSeeOther)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    if found == false {
        app.notFound(w)
        return
    }

    f := forms.New(url.Values{})
    f.Add("changeToken", token)

    app.render(w, r, "password.change.page.tmpl", &templateData{
        Form: f,
    })
}

func (app *application) passwordChange(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("email", "password", "changeToken")
    form.MatchesPattern("email", forms.EmailRX)
    form.MinLength("password", 10)

    // If there are any errors, redisplay the signup form.
    if !form.Valid() {
        app.render(w, r, "password.change.page.tmpl", &templateData{Form: form})
        return
    }

    _, err = app.users.CheckPasswordChangeToken(form.Get("email"), form.Get("changeToken"))
    if err == models.ErrTokenNotFound {
        form.Errors.Add("generic", "Token expired or does not match invite.")
        app.render(w, r, "password.change.page.tmpl", &templateData{Form: form})
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    // Update password
    _, err = app.users.UpdatePassword(form.Get("email"), form.Get("password"))
    if err != nil {
        app.serverError(w, err)
        return
    }

    _, err = app.users.MarkPasswordChangeTokenAsUsed(form.Get("changeToken"))
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", "Your password was changed successfully. Please log in.")
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

    user, _ := app.users.Get(id)
    if user.Admin || user.Translator {
        http.Redirect(w, r, "/translate", http.StatusSeeOther)
        return
    }else{
        http.Redirect(w, r, "/pair", http.StatusSeeOther)
    }
}


func (app *application) setLanguage(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get(":code")
    if code != "ES" && code != "FR" && code != "PT" && code != "SW" {
        app.notFound(w)
        return
    }

    // Add the language codes of the current user to the session, so that they are now 'logged in'.
    app.session.Put(r, "sourceLanguage", "EN")
    app.session.Put(r, "targetLanguage", code)

    user, err := app.users.Get(app.session.GetInt(r, "userID"))
    if err != nil {
        http.Redirect(w, r, "/user/login", http.StatusSeeOther)
        return
    }

    if user.Admin || user.Translator {
        http.Redirect(w, r, "/translate", http.StatusSeeOther)
        return
    }else{
        http.Redirect(w, r, "/pair", http.StatusSeeOther)
    }
}


func (app *application) logoutUser(w http.ResponseWriter, r *http.Request) {
    // Remove the userID from the session data so that the user is 'logged out'.
    app.session.Remove(r, "userID")
    app.session.Remove(r, "sourceLanguage")
    app.session.Remove(r, "targetLanguage")
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
    tmp_file, fileType := form.ProcessFileUpload(w, r, *app.maxUploadSize, *app.uploadPath, app.infoLog, app.errorLog)


    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.render(w, r, "upload.page.tmpl", &templateData{Form: form})
        return
    }

    var tpfile *files.TranslationPairFile = nil

    if fileType == "TSV" {
        tpfile = files.ReadPairsFromTsv(tmp_file)
    }
    if fileType == "XLSX" {
        tpfile = files.ReadPairsFromXlsx(tmp_file)
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

    http.Redirect(w, r, "/pair", http.StatusSeeOther)
}


func (app *application) translatePage(w http.ResponseWriter, r *http.Request) {
    limit, err := app.users.GetUserLimit(app.authenticatedUser(r).ID)
    if err != nil {
        app.serverError(w, err)
        return
    }

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()
    targetLanguage := app.session.GetString(r, "targetLanguage")

    m, err := automl.GetModelsByLanguage(app.infoLog, app.errorLog, projectId, targetLanguage)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "translate.page.tmpl",
               &templateData{Form: forms.New(nil),
                             UserLimit: limit,
                             Models: m,
                             })
}

func (app *application) translationLimitIsReached(user *models.User, text *files.TextStruct) (string, int, error) {
    limit, err := app.users.GetUserLimit(user.ID)
    if err != nil {
        return "", limit.TotalTranslated, err
    }

    if limit.TotalLimit <= limit.TotalTranslated {
        return "reached", limit.TotalTranslated, nil
    }

    if limit.TotalLimit < limit.TotalTranslated + text.CharacterCount {
        return "surpassed", limit.TotalTranslated, nil
    }

    return "ok", limit.TotalTranslated, nil
}

func (app *application) translate(w http.ResponseWriter, r *http.Request) {
    type Reply struct {
        Error           string
        CharactersUsed  string
        Translation     string
    }

    form := forms.New(r.PostForm)
    form.Required("docTitle", "sourceText", "model")

    if !form.Valid() {
        app.clientErrorDetailed(w, 400, form.Errors.ToString())
        return
    }

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")
    sourceText := files.ConvertPlainTextToTextStruct(form.Get("sourceText"))
    title := form.Get("docTitle")
    modelName := form.Get("model")

    check, charactersUsed, err := app.translationLimitIsReached(app.authenticatedUser(r), sourceText)
    if err != nil {
        app.serverError(w, err)
        return
    }

    if check != "ok"{
        reply := Reply{Error: check, CharactersUsed: fmt.Sprintf("%d", charactersUsed), Translation: ""}
        json.NewEncoder(w).Encode(reply)
        return
    }

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    targetText, err := automl.TranslateRequest(app.infoLog, app.errorLog, r, app.reports, app.users,
                                               app.authenticatedUser(r),
                                               modelName, sourceLanguage, targetLanguage, sourceText, title)

    if err != nil {
        app.serverError(w, err)
        return
    }

    limit, err := app.users.GetUserLimit(app.authenticatedUser(r).ID)
    if err != nil {
        app.serverError(w, err)
        return
    }

    targetTextString := files.ConvertTextStructToPlainText(targetText)
    reply := Reply{Error: "None", CharactersUsed: fmt.Sprintf("%d", limit.TotalTranslated), Translation: *targetTextString}
    json.NewEncoder(w).Encode(reply)
}

/*
func (app *application) translateFile(w http.ResponseWriter, r *http.Request) {
    type Reply struct {
        Error           string
        CharactersUsed  string
        Translation     string
    }

    form := forms.New(r.PostForm)
    form.Required("docTitle", "fileName", "model")
    tmp_file, fileType := form.ProcessTranslateFileUpload(w, r, *app.maxUploadSize, *app.uploadPath, app.infoLog, app.errorLog)


    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.render(w, r, "upload.page.tmpl", &templateData{Form: form})
        return
    }

    sourceText, err := files.ExtractTextToTranslateDocx(tmp_file)
    if error != nil {
        form.Errors["fileName"] = tpfile.Errors["fileName"]
        app.render(w, r, "upload.page.tmpl", &templateData{Form: form})
        return
    }

    if !form.Valid() {
        app.clientErrorDetailed(w, 400, form.Errors.ToString())
        return
    }

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")
    title := "Test"
    modelName := form.Get("model")

    check, charactersUsed, err := app.translationLimitIsReached(app.authenticatedUser(r), sourceText)
    if err != nil {
        app.serverError(w, err)
        return
    }

    if check != "ok"{
        reply := Reply{Error: check, CharactersUsed: fmt.Sprintf("%d", charactersUsed), Translation: ""}
        json.NewEncoder(w).Encode(reply)
        return
    }

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    targetText, err := automl.TranslateRequest(app.infoLog, app.errorLog, r, app.reports, app.users,
                                              app.authenticatedUser(r),
                                              modelName, sourceLanguage, targetLanguage, sourceText, title)

    if err != nil {
        app.serverError(w, err)
        return
    }

    limit, err := app.users.GetUserLimit(app.authenticatedUser(r).ID)
    if err != nil {
        app.serverError(w, err)
        return
    }

    reply := Reply{Error: "None", CharactersUsed: fmt.Sprintf("%d", limit.TotalTranslated), Translation: targetText}
    json.NewEncoder(w).Encode(reply)
}
*/

func (app *application) exportTranslation(w http.ResponseWriter, r *http.Request) {
    sourceText, ok := r.URL.Query()["source"]
    if !ok {
        app.notFound(w)
        return
    }

    targetText, ok := r.URL.Query()["target"]
    if !ok {
        app.notFound(w)
        return
    }

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    timeRequest := time.Now().Format("20060102150405")

    tmpFile := fmt.Sprintf("./tmp/translation_%s-%s_%s.docx", sourceLanguage, targetLanguage, timeRequest)

    name := fmt.Sprintf("translation_%s-%s_%s.docx", sourceLanguage, targetLanguage, timeRequest)

    size := files.WriteTranslationInterleavedToDocx(tmpFile, sourceText[0], targetText[0])

    app.downloadFile(w, r, "docx", tmpFile, name, size)
}

func (app *application) showModels(w http.ResponseWriter, r *http.Request) {

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    m, err := automl.ListModelsRequest(app.infoLog, app.errorLog, projectId)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "show.model.page.tmpl", &templateData{Models: m})
}

func (app *application) showTrainingStatus(w http.ResponseWriter, r *http.Request) {

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    trainReport, err := automl.ListTrainOperationsRequest(app.infoLog, app.errorLog, projectId)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "train.status.page.tmpl", &templateData{TrainReport: trainReport})
}


func (app *application) showDatasets(w http.ResponseWriter, r *http.Request) {

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    d, err := automl.ListDatasetsRequest(app.infoLog, app.errorLog, projectId)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.render(w, r, "show.dataset.page.tmpl", &templateData{Datasets: d})
}


func (app *application) initValidatePair(w http.ResponseWriter, r *http.Request) {
    bookId, err := strconv.Atoi(r.URL.Query().Get(":bookid"))
    if err != nil || bookId < 1 {
        app.notFound(w)
        return
    }

    b, err := app.pairs.GetBook(bookId)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    chapterId, err := strconv.Atoi(r.URL.Query().Get(":chapterid"))
    if err != nil || chapterId < 1 || chapterId > b.Chapter {
        app.notFound(w)
        return
    }

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    newId, err := app.pairs.GetNewIDToValidate(sourceLanguage, targetLanguage, bookId, chapterId)
    if err == models.ErrNoRecord {
        app.session.Put(r, "flash", fmt.Sprintf("No pairs to validate found in Book %s and chapter %d",
                                                b.Name, chapterId))
        http.Redirect(w, r, "/pair", http.StatusSeeOther)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    http.Redirect(w, r, fmt.Sprintf("/pair/validate/%d", newId), http.StatusSeeOther)
}


func (app *application) validateSelectedPairs(w http.ResponseWriter, r *http.Request) {
    bookId, err := strconv.Atoi(r.URL.Query().Get(":bookid"))
    if err != nil || bookId < 1 {
        app.notFound(w)
        return
    }

    b, err := app.pairs.GetBook(bookId)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    chapterId, err := strconv.Atoi(r.URL.Query().Get(":chapterid"))
    if err != nil || chapterId < 1 || chapterId > b.Chapter {
        app.notFound(w)
        return
    }

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    form := forms.New(r.PostForm)
    form.Required("idList")

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.session.Put(r, "flash", "No pairs were selected!")

        http.Redirect(w, r, fmt.Sprintf("/pair/book/%d/chapter/%d", bookId, chapterId), http.StatusSeeOther)
        return
    }

    err = app.pairs.ValidateSelectedPairsFromChapter(sourceLanguage, targetLanguage, bookId, chapterId, form.Get("idList"))
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", "Selected pairs validated for this chapter")
    http.Redirect(w, r, fmt.Sprintf("/pair/book/%d/chapter/%d", bookId, chapterId), http.StatusSeeOther)
}


func (app *application) unvalidateSelectedPairs(w http.ResponseWriter, r *http.Request) {
    bookId, err := strconv.Atoi(r.URL.Query().Get(":bookid"))
    if err != nil || bookId < 1 {
        app.notFound(w)
        return
    }

    b, err := app.pairs.GetBook(bookId)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    chapterId, err := strconv.Atoi(r.URL.Query().Get(":chapterid"))
    if err != nil || chapterId < 1 || chapterId > b.Chapter {
        app.notFound(w)
        return
    }

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    form := forms.New(r.PostForm)
    form.Required("idList")

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.session.Put(r, "flash", "No pairs were selected!")

        http.Redirect(w, r, fmt.Sprintf("/pair/book/%d/chapter/%d", bookId, chapterId), http.StatusSeeOther)
        return
    }

    err = app.pairs.UnvalidateSelectedPairsFromChapter(sourceLanguage, targetLanguage, bookId, chapterId, form.Get("idList"))
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", "Selected pairs unvalidated for this chapter")
    http.Redirect(w, r, fmt.Sprintf("/pair/book/%d/chapter/%d", bookId, chapterId), http.StatusSeeOther)
}


func (app *application) validatePair(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.URL.Query().Get(":id"))
    if err != nil || id < 1 {
        app.notFound(w)
        return
    }

    err = r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("validate")

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.session.Put(r, "flash", "Invalid form!")
        http.Redirect(w, r, fmt.Sprintf("/pair/%d", id), http.StatusSeeOther)
        return
    }


    err = app.pairs.Validate(id)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", fmt.Sprintf("Pair %d successfully validated!", id))
    http.Redirect(w, r, fmt.Sprintf("/pair/%d", id), http.StatusSeeOther)
}


func (app *application) unvalidatePair(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.URL.Query().Get(":id"))
    if err != nil || id < 1 {
        app.notFound(w)
        return
    }

    err = r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("unvalidate")

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        app.session.Put(r, "flash", "Invalid form!")
        http.Redirect(w, r, fmt.Sprintf("/pair/%d", id), http.StatusSeeOther)
        return
    }

    err = app.pairs.Unvalidate(id)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", fmt.Sprintf("Pair %d successfully unvalidated!", id))
    http.Redirect(w, r, fmt.Sprintf("/pair/edit/%d", id), http.StatusSeeOther)
}


func (app *application) deleteDatasetForm(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    v := url.Values{}
    v.Add("name", id)

    app.render(w, r, "delete.dataset.page.tmpl", &templateData{Form: forms.New(v)})
}


func (app *application) deleteDataset(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    form := forms.New(r.PostForm)
    form.OneRequired("yes", "no")

    if !form.Valid() {
        app.render(w, r, "show.dataset.page.tmpl", &templateData{Form: form})
        return
    }

    if form.Get("yes") != "" {
        err = automl.DeleteDatasetRequest(app.infoLog, app.errorLog, projectId, id)
    }
    if err != nil {
        app.serverError(w, err)
        return
    }

    if form.Get("yes") != ""{
        app.session.Put(r, "flash", "Dataset successfully deleted!")
    }else{
        app.session.Put(r, "flash", "Dataset not deleted!")
    }

    http.Redirect(w, r, fmt.Sprintf("/dataset"), http.StatusSeeOther)
}


func (app *application) trainDatasetForm(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    m, err := automl.ListModelsRequest(app.infoLog, app.errorLog, projectId)
    if err != nil {
        app.serverError(w, err)
        return
    }

    v := url.Values{}
    v.Add("datasetDisplayName", id)
    v.Add("datasetName", id)

    app.render(w, r, "train.dataset.page.tmpl", &templateData{
        Form: forms.New(v),
        Models: m,
    })
}


func (app *application) trainDataset(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    form := forms.New(r.PostForm)
    form.OneRequired("train", "cancel")

    if form.Get("cancel") != "" {
        app.session.Put(r, "flash", "Training not launched!")
        http.Redirect(w, r, fmt.Sprintf("/dataset"), http.StatusSeeOther)
        return
    }

    form.Required("modelDisplayName")
    // Max number of chars for text input
    maxChar := 10000
    form.MaxLength("modelDisplayName", maxChar)
    // Languages codes to check

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    m, err := automl.ListModelsRequest(app.infoLog, app.errorLog, projectId)
    if err != nil {
        app.serverError(w, err)
        return
    }

    // Avoid taking an already existing model name
    form.NotPermittedValues("modelDisplayName", automl.GetModelsDisplayName(m)...)

    if !form.Valid() {
        v := url.Values{}
        v.Add("datasetDisplayName", id)
        v.Add("datasetName", id)

        app.render(w, r, "train.dataset.page.tmpl", &templateData{
            Form: forms.New(v),
            Models: m,
        })
        return
    }

    // We send the train request
    err = automl.TrainModelRequest(app.infoLog, app.errorLog, projectId, id, form.Get("modelDisplayName"))
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.session.Put(r, "flash", "Training launched successfully!")
    http.Redirect(w, r, fmt.Sprintf("/train/status"), http.StatusSeeOther)
}


func (app *application) deleteModelForm(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    f := forms.New(url.Values{})
    f.Add("name", id)

    app.render(w, r, "delete.model.page.tmpl", &templateData{Form: f})
}


func (app *application) deleteModel(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    form := forms.New(r.PostForm)
    form.OneRequired("yes", "no")

    if !form.Valid() {
        app.render(w, r, "show.model.page.tmpl", &templateData{Form: form})
        return
    }

    if form.Get("yes") != "" {
        err = automl.DeleteModelRequest(app.infoLog, app.errorLog, projectId, id)
    }
    if err != nil {
        app.serverError(w, err)
        return
    }

    if form.Get("yes") != ""{
        app.session.Put(r, "flash", "Model successfully deleted!")
    }else{
        app.session.Put(r, "flash", "Model not deleted!")
    }

    http.Redirect(w, r, fmt.Sprintf("/model"), http.StatusSeeOther)
}


func (app *application) cancelTrainingOperationForm(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    f := forms.New(url.Values{})
    f.Add("name", id)

    app.render(w, r, "cancel.train.page.tmpl", &templateData{Form: f})
}


func (app *application) cancelTrainingOperation(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get(":name")
    if id == "" {
        app.notFound(w)
        return
    }

    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        app.serverError(w, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    projectId := scanner.Text()

    form := forms.New(r.PostForm)
    form.OneRequired("yes", "no")

    if !form.Valid() {
        app.render(w, r, "show.model.page.tmpl", &templateData{Form: form})
        return
    }

    if form.Get("yes") != "" {
        err = automl.CancelTrainRequest(app.infoLog, app.errorLog, projectId, id)
    }
    if err != nil {
        app.serverError(w, err)
        return
    }

    if form.Get("yes") != ""{
        app.session.Put(r, "flash", "Train operation successfully cancelled!")
    }else{
        app.session.Put(r, "flash", "Train operation not cancelled!")
    }

    http.Redirect(w, r, fmt.Sprintf("/train/status"), http.StatusSeeOther)
}


func (app *application) exportValidatedPairsForm(w http.ResponseWriter, r *http.Request) {
    bookId, err := strconv.Atoi(r.URL.Query().Get(":bookid"))
    if err != nil || bookId < 1 {
        app.notFound(w)
        return
    }

    b, err := app.pairs.GetBook(bookId)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    chapterId, err := strconv.Atoi(r.URL.Query().Get(":chapterid"))
    if err != nil || chapterId < 1 || chapterId > b.Chapter {
        app.notFound(w)
        return
    }

    b.Chapter = chapterId

    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    p, err := app.pairs.GetValidatedNotExportedFromChapter(sourceLanguage, targetLanguage, b.ID, b.Chapter)
    if err != nil {
        app.serverError(w, err)
        return
    }

    if len(p) == 0 {
        app.session.Put(r, "flash", "No pairs available to be exported!")
        redirectUrl := fmt.Sprintf("/pair/book/%d/chapter/%d", b.ID, b.Chapter)
        http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
        return
    }

    app.render(w, r, "export.pair.page.tmpl", &templateData{
        Pairs: p,
        Book: b,
        Form: forms.New(nil),
    })
}


func (app *application) exportAllValidatedPairsForm(w http.ResponseWriter, r *http.Request) {
    sourceLanguage := app.session.GetString(r, "sourceLanguage")
    targetLanguage := app.session.GetString(r, "targetLanguage")

    p, err := app.pairs.GetValidatedNotExported(sourceLanguage, targetLanguage)
    if err != nil {
        app.serverError(w, err)
        return
    }

    if len(p) == 0 {
        app.session.Put(r, "flash", "No pairs available to be exported!")
        redirectUrl := "/pair"
        http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
        return
    }

    app.render(w, r, "export.all.pair.page.tmpl", &templateData{
        Pairs: p,
        Form: forms.New(nil),
    })
}


func (app *application) downloadFile(w http.ResponseWriter, r *http.Request, fileType, tmpFile, name, fileSize string) {
    if fileType == "tsv"{
        w.Header().Set("Content-Type", "text/tab-separated-values")
    } else if fileType == "docx"{
        w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
    } else if fileType == "zip"{
        w.Header().Set("Content-Type", "application/zip")
    }  else if fileType == "tar"{
        w.Header().Set("Content-Type", "application/x-tar")
    }
    w.Header().Set("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", name))
    w.Header().Set("Content-Length", fileSize)
    http.ServeFile(w, r, tmpFile)
}


func (app *application) exportValidatedPairs(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("name", "idList")

    bookId, err := strconv.Atoi(form.Get("book"))
    if err != nil || bookId < 1 {
        app.notFound(w)
        return
    }

    b, err := app.pairs.GetBook(bookId)
    if err == models.ErrNoRecord {
        app.notFound(w)
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    chapterId, err := strconv.Atoi(form.Get("chapter"))
    if err != nil || chapterId < 1 || chapterId > b.Chapter {
        app.notFound(w)
        return
    }

    b.Chapter = chapterId

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        sourceLanguage := app.session.GetString(r, "sourceLanguage")
        targetLanguage := app.session.GetString(r, "targetLanguage")

        p, err := app.pairs.GetValidatedNotExportedFromChapter(sourceLanguage, targetLanguage, b.ID, b.Chapter)
        if err != nil {
            app.serverError(w, err)
            return
        }

        app.render(w, r, "export.pair.page.tmpl", &templateData{
            Pairs: p,
            Book: b,
            Form: form,
        })
        return
    }

    pairs, err := app.pairs.GetAndMarkedExported(app.infoLog, app.errorLog, form.Get("idList"), form.Get("name"))
    if err == models.ErrDuplicateDataset {
        sourceLanguage := app.session.GetString(r, "sourceLanguage")
        targetLanguage := app.session.GetString(r, "targetLanguage")

        p, err := app.pairs.GetValidatedNotExported(sourceLanguage, targetLanguage)
        if err != nil {
            app.serverError(w, err)
            return
        }

        form.Errors.Add("name", "Dataset name already used")

        app.render(w, r, "export.pair.page.tmpl", &templateData{
            Pairs: p,
            Form: form,
        })
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    tmpName := fmt.Sprintf("dataset_%s", time.Now().Format("20060102150405"))
    tmpFile := fmt.Sprintf("./tmp/%s.tsv", tmpName)
    fileSize := files.WriteDataset(tmpFile, pairs)

    app.session.Put(r, "flash", "Dataset successfully exported!")
    app.downloadFile(w, r, "tsv", tmpFile, form.Get("name"), fileSize)
}


func (app *application) exportAllValidatedPairs(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form := forms.New(r.PostForm)
    form.Required("name", "idList")

    // If the form isn't valid, redisplay the template passing in the
    // form.Form object as the data.
    if !form.Valid() {
        sourceLanguage := app.session.GetString(r, "sourceLanguage")
        targetLanguage := app.session.GetString(r, "targetLanguage")

        p, err := app.pairs.GetValidatedNotExported(sourceLanguage, targetLanguage)
        if err != nil {
            app.serverError(w, err)
            return
        }

        app.render(w, r, "export.all.pair.page.tmpl", &templateData{
            Pairs: p,
            Form: form,
        })
        return
    }

    pairs, err := app.pairs.GetAndMarkedExported(app.infoLog, app.errorLog, form.Get("idList"), form.Get("name"))
    if err == models.ErrDuplicateDataset {
        sourceLanguage := app.session.GetString(r, "sourceLanguage")
        targetLanguage := app.session.GetString(r, "targetLanguage")

        p, err := app.pairs.GetValidatedNotExported(sourceLanguage, targetLanguage)
        if err != nil {
            app.serverError(w, err)
            return
        }

        form.Errors.Add("name", "Dataset name already used")

        app.render(w, r, "export.all.pair.page.tmpl", &templateData{
            Pairs: p,
            Form: form,
        })
        return
    } else if err != nil {
        app.serverError(w, err)
        return
    }

    tmpName := fmt.Sprintf("dataset_%s", time.Now().Format("20060102150405"))
    tmpFile := fmt.Sprintf("./tmp/%s.tsv", tmpName)
    fileSize := files.WriteDataset(tmpFile, pairs)

    app.session.Put(r, "flash", "Dataset successfully exported!")
    app.downloadFile(w, r, "tsv", tmpFile, form.Get("name"), fileSize)
}