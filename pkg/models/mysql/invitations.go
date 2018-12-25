package mysql

import (
    "database/sql"
    "strings"
    "time"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    //"github.com/go-sql-driver/mysql"
    "golang.org/x/crypto/bcrypt"
)

type InvitationModel struct {
    DB *sql.DB
}

func (m *InvitationModel) Insert(email, role string) (*models.Invitation, error) {
    stmt := `SELECT count(*) FROM users WHERE email = ?`

    var count int
    row := m.DB.QueryRow(stmt, email)
    err := row.Scan(&count)
    if err != nil {
        return nil, err
    }
    if count != 0{
        return nil, models.ErrDuplicateEmail
    }

    seed := strings.Join([]string{email, time.Now().String()}, "")

    token, err := bcrypt.GenerateFromPassword([]byte(seed), 12)
    if err != nil {
        return nil, err
    }

    stmt = `INSERT INTO user_invitation (email, token, role, created, expires)
    VALUES(?, ?, ?, UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL 1 DAY))`

    result, err := m.DB.Exec(stmt, email, token, role)
    if err != nil {
        return nil, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return nil, err
    }

    return m.Get(int(id))
}


func (m *InvitationModel) CheckToken(email, token string) (*models.Invitation, error) {
    stmt := `SELECT id FROM user_invitation 
    WHERE token = ? AND email= ? AND UTC_TIMESTAMP() <= expires`

    var id int
    row := m.DB.QueryRow(stmt, token, email)
    err := row.Scan(&id)
    if err == sql.ErrNoRows {
        return nil, models.ErrTokenNotFound
    } else if err != nil {
        return nil, err
    }
    return m.Get(int(id))
}


func (m *InvitationModel) TokenExists(token string) (bool, error) {
    stmt := `SELECT id FROM user_invitation 
    WHERE token = ? AND UTC_TIMESTAMP() <= expires`

    var id int
    row := m.DB.QueryRow(stmt, token)
    err := row.Scan(&id)
    if err == sql.ErrNoRows {
        return false, models.ErrTokenNotFound
    } else if err != nil {
        return false, err
    }
    return true, nil
}


func (m *InvitationModel) Get(id int) (*models.Invitation, error) {
    i := &models.Invitation{}

    stmt := `SELECT id, email, role, token, created, expires, used
             FROM user_invitation WHERE id = ?`
    row := m.DB.QueryRow(stmt, id)
    err := row.Scan(&i.ID, &i.Email, &i.Role, &i.Token, &i.Created, &i.Expires, &i.Used)
    if err == sql.ErrNoRows {
        return nil, models.ErrTokenNotFound
    } else if err != nil {
        return nil, err
    }

    return i, nil
}