package reports

import (
    "encoding/base64"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
    "github.com/alejandrosame/gcp-mt-utils/pkg/models/mysql"

    "github.com/sendgrid/sendgrid-go"
    "github.com/sendgrid/sendgrid-go/helpers/mail"
)

func SendEmail(infoLog, errorLog *log.Logger, r *http.Request, reportModel *mysql.ReportModel, user *models.User,
               characterCount int, requestDate time.Time, title, filePath string) {

    requestDateFormatted := requestDate.Format("2006/01/02-15:04:05")
    senderReceiverMap, err := reportModel.GetSenderReceiver()
    if err != nil {
        errorLog.Println(err)
    }

    from := mail.NewEmail((*senderReceiverMap)["Sender"]["Name"], (*senderReceiverMap)["Sender"]["Email"])
    subject := fmt.Sprintf("[Report] Translation request from %s", user.Email)
    to := mail.NewEmail((*senderReceiverMap)["Receiver"]["Name"], (*senderReceiverMap)["Receiver"]["Email"])

    plainTextContent, htmlContent, err := GenerateReportFromRequest(infoLog, errorLog, r, user,
                                                                    characterCount, title, requestDateFormatted)

    if err != nil {
        errorLog.Println(err)
    }

    attachmentFile := mail.NewAttachment()
    dat, err := ioutil.ReadFile(filePath)
    if err != nil {
        errorLog.Println(err)
    }
    encoded := base64.StdEncoding.EncodeToString([]byte(dat))
    attachmentFile.SetContent(encoded)
    attachmentFile.SetFilename(fmt.Sprintf("%s.docx", title))
    attachmentFile.SetDisposition("attachment")

    message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
    message.AddAttachment(attachmentFile)

    client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
    response, err := client.Send(message)
    if err != nil {
        errorLog.Println(err)
    } else{
        infoLog.Println(response.StatusCode)
        infoLog.Println(response.Body)
        infoLog.Println(response.Headers)
    }
}