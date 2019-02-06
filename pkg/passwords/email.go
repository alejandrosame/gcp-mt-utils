package passwords

import (
    "fmt"
    "log"
    "os"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
    "github.com/alejandrosame/gcp-mt-utils/pkg/models/mysql"

    "github.com/sendgrid/sendgrid-go"
    "github.com/sendgrid/sendgrid-go/helpers/mail"
)

func SendEmail(infoLog, errorLog *log.Logger, reportModel *mysql.ReportModel, name string, passwordChangeRequest *models.PasswordChangeRequest) {

    senderReceiverMap, err := reportModel.GetSenderReceiver()
    if err != nil {
        errorLog.Println(err)
    }

    from := mail.NewEmail((*senderReceiverMap)["Sender"]["Name"], (*senderReceiverMap)["Sender"]["Email"])
    subject := fmt.Sprintf("Password change request for secondride")
    to := mail.NewEmail(name, passwordChangeRequest.Email)

    plainTextContent := fmt.Sprintf(`Hello, %s!

    	Please, use the next link to update your password:


    https://secondride.org/user/password/change?token=%s

    This token will expire in 10 minutes`,
    name, passwordChangeRequest.Token)

    htmlContent := fmt.Sprintf(`Hello, %s!<br><br>Please, use the next link to update your password:<br><br>
    <a>https://secondride.org/user/password/change?token=%s</a><br><br>
    <strong>This token will expire in 10 minutes</strong>.`,
    name, passwordChangeRequest.Token)

    message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

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