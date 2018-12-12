package models

import (
    "errors"
    "time"
)

var (
    ErrNoRecord = errors.New("models: no matching record found")
    ErrInvalidCredentials = errors.New("models: invalid credentials")
    ErrDuplicateEmail = errors.New("models: duplicate email")
)

// Misc models
type ValidationStats struct {
    Validated       int
    NotValidated    int
    Total           int
    Percent         float64
}


// Models for input/output files
type FilePair struct {
    SourceLanguage  string
    TargetLanguage  string
    SourceText      string
    TargetText      string
}

// Models for DB objects
type Pair struct {
    ID              int
    FilePair
    Created         time.Time
}

type User struct {
    ID             int
    Name           string
    Email          string
    HashedPassword []byte
    Created        time.Time
}