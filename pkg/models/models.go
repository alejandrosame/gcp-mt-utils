package models

import (
    "database/sql"
    "errors"
    "time"
)

var (
    ErrNoRecord = errors.New("models: no matching record found")
    ErrInvalidCredentials = errors.New("models: invalid credentials")
    ErrDuplicateEmail = errors.New("models: duplicate email")
    ErrTokenNotFound = errors.New("models: token not found, expired or does not match with email")
    ErrDuplicateDataset = errors.New("models: duplicate dataset")
    ErrInjection = errors.New("models: input is not the expected")
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
    SourceVersion   string
    TargetLanguage  string
    TargetVersion   string
    Detail          string
    SourceText      string
    TargetText      string
}

// Models for DB objects
type Pair struct {
    ID              int
    FilePair
    Comments        sql.NullString
    Validated       bool
    GcpDataset      sql.NullString
    Created         time.Time
    Updated         time.Time
}

type User struct {
    ID             int
    Name           string
    Email          string
    HashedPassword []byte
    Created        time.Time
    Super          bool
    Admin          bool
    Validator      bool
    Translator     bool
}


type Invitation struct {
    ID             int
    Token          []byte
    Role           string
    Email          string
    Created        time.Time
    Expires        time.Time
    Used           bool
}