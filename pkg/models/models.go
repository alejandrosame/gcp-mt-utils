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
    ErrEmailNotFound = errors.New("models: email not found")
    ErrTokenNotFound = errors.New("models: token not found, expired or did not match user email")
    ErrTokenStillValid = errors.New("models: previous token still valid.")
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

type RoleLimit struct {
    UserRole       string
    CharacterLimit int
}

type UserLimit struct {
    ID              int
    Super           bool
    Admin           bool
    Validator       bool
    Translator      bool
    Name            string
    Email           string
    CharacterLimit  int
    TotalLimit      int
    TotalTranslated int
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

type PasswordChangeRequest struct {
    ID             int
    Token          []byte
    Email          string
    Created        time.Time
    Expires        time.Time
    Used           bool
}

type BibleBook struct {
    ID              int
    Name            string
    Chapter         int
    Testament       string
    Stats           *ValidationStats
    ChapterStats    []*ValidationStats
}