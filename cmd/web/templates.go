package main

import (
    "bufio"
    "fmt"
    "html/template"
    "io/ioutil"
    "math/rand"
    "os"
    "path/filepath"
    "strings"
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
    Book              *models.BibleBook
    NextChapter       *models.BibleBook
    Books             []*models.BibleBook
    Models            []*automl.Model
    TrainReport       *automl.TrainOperationReport
    Datasets          []*automl.Dataset
    ValidationStats   *models.ValidationStats
    SignUpInvitation  *models.Invitation
    Languages         string
    RoleLimit         *models.RoleLimit
    UserLimit         *models.UserLimit
    AllUserLimits     []*models.UserLimit
}


func humanDate(t time.Time) string {
    return t.Format("02 Jan 2006 at 15:04")
}

func last(s string) string {
    temp := strings.Split(s, "/")
    return temp[len(temp)-1]
}

func tokenToString(b []byte) string {
    return string(b[:60])
}

func truncate(s string, count int) string {
    r := []rune(s)
    m := count
    ellipsis := "..."
    if len(r) < m {
        m = len(r)
        ellipsis = ""
    }
    return fmt.Sprintf("%s%s", string(r[:m]), ellipsis)
}

func getProject() string {
    file, err := os.Open("./auth/auth.txt")
    if err != nil {
        return ""
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Scan()
    scanner.Scan()
    return scanner.Text()
}

func languageSource(s string) string {
    temp := strings.Split(s, " - ")
    return temp[0]
}

func languageTarget(s string) string {
    temp := strings.Split(s, " - ")
    return temp[1]
}

func rangeInt(start, end int) (stream chan int) {
    stream = make(chan int)
    go func() {
        for i := start; i <= end; i++ {
            stream <- i
        }
        close(stream)
    }()
    return
}

func ShuffleStrings(vals []string) []string {
  r := rand.New(rand.NewSource(time.Now().Unix()))
  ret := make([]string, len(vals))
  perm := r.Perm(len(vals))
  for i, randIndex := range perm {
    ret[i] = vals[randIndex]
  }
  return ret
}

func ShuffleFiles(vals []os.FileInfo) []os.FileInfo {
  r := rand.New(rand.NewSource(time.Now().Unix()))
  ret := make([]os.FileInfo, len(vals))
  perm := r.Perm(len(vals))
  for i, randIndex := range perm {
    ret[i] = vals[randIndex]
  }
  return ret
}

func rangeFlags(language string, max int) (stream chan string) {
    m := make(map[string][]string)
    m["ES"] = ShuffleStrings([]string{"es", "ar", "bo", "cl", "co", "cr", "cu", "do", "ec", "sv", "gt", "hn", "mx", "ni", "pa", "py",
                       "pe", "uy", "ve", "gq"})
    m["FR"] = ShuffleStrings([]string{"cd", "fr", "ca", "mg", "cm", "ci", "ne", "bf", "ml", "sn", "td", "gn", "rw", "be", "bi", "bj",
                       "ht", "ch", "tg", "cf", "cg", "ga", "gq", "dj", "km", "lu", "vu", "sc", "mc"})
    m["PT"] = ShuffleStrings([]string{"br", "ao", "mz", "pt", "gw", "tl", "gq", "cv", "st"})
    m["SW"] = ShuffleStrings([]string{"tz", "cd", "ke", "so", "mz", "bi", "ug", "km", "zm", "mw", "mg"})

    stream = make(chan string)
    go func() {
        limit := len(m[language])
        if max < limit {
            limit = max
        }

        for i := 0; i < limit; i++ {
            stream <- fmt.Sprintf("/static/img/flags/%s.png", m[language][i])
        }
        close(stream)
    }()
    return
}

func rangePeople(language string, max int) (stream chan string) {
    stream = make(chan string)
    go func() {
        files, err := ioutil.ReadDir(fmt.Sprintf("./ui/static/img/people/%s", strings.ToLower(language)))
        if err != nil {
            stream <- ""
        } else {
            limit := len(files)
            if max < limit {
                limit = max
            }

            files = ShuffleFiles(files[:limit])

            for _, f := range files {
                stream <- fmt.Sprintf("/static/img/people/%s/%s", strings.ToLower(language), f.Name())
            }
        }
        close(stream)
    }()
    return
}

func minus(a, b int) (int) {
    return a-b
}


// Initialize a template.FuncMap object and store it in a global variable. This is
// essentially a string-keyed map which acts as a lookup between the names of our
// custom template functions and the functions themselves.
var functions = template.FuncMap{
    "humanDate": humanDate,
    "last": last,
    "getProject": getProject,
    "tokenToString": tokenToString,
    "truncate": truncate,
    "languageSource": languageSource,
    "languageTarget": languageTarget,
    "rangeInt": rangeInt,
    "rangeFlags": rangeFlags,
    "rangePeople": rangePeople,
    "minus": minus,
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