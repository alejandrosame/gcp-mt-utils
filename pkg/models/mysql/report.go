package mysql

import (
    "database/sql"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)


type ReportModel struct {
    DB *sql.DB
}


func (r *ReportModel) GetSenderReceiver() (*map[string]map[string]string, error) {
    reportEmailUsers := make(map[string]map[string]string)

    sender := map[string]string{}
    receiver := map[string]string{}

    email := ""
    name := ""

    stmt := `SELECT email, user_name
             FROM report_email_sender`
    err := r.DB.QueryRow(stmt).Scan(&email, &name)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    sender["Name"] = name
    sender["Email"] = email


    stmt = `SELECT email, user_name
             FROM report_email_receiver`
    err = r.DB.QueryRow(stmt).Scan(&email, &name)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    receiver["Name"] = name
    receiver["Email"] = email

    reportEmailUsers["Sender"] = sender
    reportEmailUsers["Receiver"] = receiver

    return &reportEmailUsers, nil
}