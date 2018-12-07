package forms

import (
    "crypto/rand"
    "fmt"
    "io/ioutil"
    "log"
    "mime"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "unicode/utf8"
)

var EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
// TODO: Add regexp for valid characters in each language. E.g. regexp.MatchString("^[a-zA-Z1-9\\-]+$")


// Create a custom Form struct, which anonymously embeds a url.Values object
// (to hold the form data) and an Errors field to hold any validation errors
// for the form data.
type Form struct {
    url.Values
    Errors errors
}

func New(data url.Values) *Form {
    return &Form{
        data,
        errors(map[string][]string{}),
    }
}


func (f *Form) Required(fields ...string) {
    for _, field := range fields {
        value := f.Get(field)
        if strings.TrimSpace(value) == "" {
            f.Errors.Add(field, "This field cannot be blank")
        }
    }
}


func (f *Form) MaxLength(field string, d int) {
    value := f.Get(field)
    if value == "" {
        return
    }
    if utf8.RuneCountInString(value) > d {
        f.Errors.Add(field, fmt.Sprintf("This field is too long (maximum is %d characters)", d))
    }
}


func (f *Form) PermittedValues(field string, opts ...string) {
    value := f.Get(field)
    if value == "" {
        return
    }
    for _, opt := range opts {
        if value == opt {
            return
        }
    }
    f.Errors.Add(field, "This field is invalid")
}


func (f *Form) MinLength(field string, d int) {
    value := f.Get(field)
    if value == "" {
        return
    }
    if utf8.RuneCountInString(value) < d {
        f.Errors.Add(field, fmt.Sprintf("This field is too short (minimum is %d characters)", d))
    }
}


func (f *Form) MatchesPattern(field string, pattern *regexp.Regexp) {
    value := f.Get(field)
    if value == "" {
        return
    }
    if !pattern.MatchString(value) {
        f.Errors.Add(field, "This field is invalid")
    }
}


func (f *Form) Valid() bool {
    return len(f.Errors) == 0
}


func (f *Form) ProcessFileUpload(w http.ResponseWriter, r *http.Request, maxUploadSize int64, uploadPath string,
                                 infoLog *log.Logger, errorLog *log.Logger) (string, string) {

    r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
    if err := r.ParseMultipartForm(maxUploadSize); err != nil {
        f.Errors.Add("fileName", fmt.Sprint("This file is too big (max %d MB)", maxUploadSize))
        return "", ""
    }

    fileType := r.PostFormValue("type")
    file, _, err := r.FormFile("fileName")
    if err != nil {
        f.Errors.Add("fileName", "Please, select a file to upload")
        return "", ""
    }
    defer file.Close()

    fileBytes, err := ioutil.ReadAll(file)
    if err != nil {
        f.Errors.Add("fileName", "This file is invalid")
        return "", ""
    }

    filetype := http.DetectContentType(fileBytes)
    if filetype != "application/zip" &&
       filetype != "text/plain; charset=utf-8" {
        errorLog.Printf("INVALID_FILE_TYPE: %s", filetype)
        f.Errors.Add("fileName", "File type not supported (supported types are TSV and XLSX)")
        return "", ""
    }

    fileName := randToken(12)
    fileEndings, err := mime.ExtensionsByType(fileType)
    if err != nil {
        if filetype == "text/plain; charset=utf-8" {
            infoLog.Printf("CANT_READ_FILE_TYPE. Assuming TSV file type")
            fileType = "TSV"
            fileEndings = append(fileEndings, ".tsv")
        }
        if filetype == "application/zip" {
            infoLog.Printf("CANT_READ_FILE_TYPE. Assuming XLSX file type")
            fileType = "XLSX"
            fileEndings = append(fileEndings, ".xlsx")
        }
    }

    newPath := filepath.Join(uploadPath, fileName+fileEndings[0])
    infoLog.Printf("FileType: %s, File: %s\n", fileType, newPath)

    newFile, err := os.Create(newPath)
    if err != nil {
        f.Errors.Add("fileName", "Cannot create this file type in temporary folder")
        return "", ""
    }
    defer newFile.Close()
    if _, err := newFile.Write(fileBytes); err != nil {
        f.Errors.Add("fileName", "Cannot write this file type in temporary folder")
        return "", ""
    }

    return newPath, fileType
}


func randToken(len int) string {
    b := make([]byte, len)
    rand.Read(b)
    return fmt.Sprintf("%x", b)
}