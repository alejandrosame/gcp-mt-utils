package reports

import (
	"encoding/base64"
  	"io/ioutil"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func SendEmail(infoLog, errorLog *log.Logger, r *http.Request, characterCount int, fileName, filePath string) {
	from := mail.NewEmail("xxxx", "xxx")
	subject := "[Test][Report] Translation request on XXX"
	to := mail.NewEmail("xxx", "xxx")

	plainTextContent, htmlContent := GenerateReportFromRequest(infoLog, errorLog, r, characterCount)
	plainTextContent := "Still testing without actual user info"
	htmlContent := "<strong>Still testing without actual user info</strong>"
	
	attachmentFile := mail.NewAttachment()
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		errorLog.Println(err)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(dat))
	attachmentFile.SetContent(encoded)
	attachmentFile.SetFilename(fileName)
	attachmentFile.SetDisposition("attachment")

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	message.AddAttachment(attachmentFile)

	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	_, err = client.Send(message)
	if err != nil {
		errorLog.Println(err)
	}
}