package models

import (
    "errors"
    "time"
)

var ErrNoRecord = errors.New("models: no matching record found")

type Pair struct {
    ID      		int
    SourceLanguage  string
    TargetLanguage  string
    SourceText   	string
    TargetText   	string
    Created 		time.Time
}