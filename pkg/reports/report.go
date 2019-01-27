package reports

import (
    "fmt"
    "log"
    "net/http"
    "sort"
    "strings"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

func GenerateReportFromRequest(infoLog, errorLog *log.Logger, r *http.Request, user *models.User,
                               characterCount int, title, requestDate string) (string, string){

    report := map[string]string{}

    report["User Name"] = user.Name
    report["User Email"] = user.Email
    report["Device"] = r.Header.Get("User-Agent")
    report["Location"] = strings.Split(r.RemoteAddr, ":")[0]

    report["Title"] = title
    report["Translation Request Date"] = requestDate
    report["Character Count"] = fmt.Sprintf("%d", characterCount)


    plainContent := ""
    htmlContent := ""

    keys := make([]string, 0)
    for k, _ := range report {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    for _, k := range keys {
        plainContent += fmt.Sprintf("-%s: %s\n", k, report[k])
        htmlContent += fmt.Sprintf("<strong>-%s: </strong>%s<br>", k, report[k])
    }

    return strings.TrimRight(plainContent, "\n"), strings.TrimRight(htmlContent, "\n")
}