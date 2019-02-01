package forms

import (
    "fmt"
    "strings"
)

// Define a new errors type, which we will use to hold the validation error
// messages for forms. The name of the form field will be used as the key in
// this map.
type errors map[string][]string

func (e errors) Add(field, message string) {
    e[field] = append(e[field], message)
}


func (e errors) Get(field string) string {
    es := e[field]
    if len(es) == 0 {
        return ""
    }
    return es[0]
}

func (e errors) ToString() string {
    errorString := ""

    for k, v := range e {
        for _, element := range v {
            errorString = fmt.Sprintf("%s[%s]: %s\n", errorString, k, element);
        }
    }

    return strings.TrimRight(errorString, "\n")
}
