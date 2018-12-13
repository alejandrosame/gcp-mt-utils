package mysql

import (
    "database/sql"
    "strings"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    "github.com/go-sql-driver/mysql"
    "golang.org/x/crypto/bcrypt"
)

type UserModel struct {
    DB *sql.DB
}

func (m *UserModel) Insert(name, email, password string) error {
    // Create a bcrypt hash of the plain-text password.
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return err
    }

    stmt := `INSERT INTO users (name, email, hashed_password, created)
    VALUES(?, ?, ?, UTC_TIMESTAMP())`

    _, err = m.DB.Exec(stmt, name, email, string(hashedPassword))
    if err != nil {
        if mysqlErr, ok := err.(*mysql.MySQLError); ok {
            if mysqlErr.Number == 1062 && strings.Contains(mysqlErr.Message, "users_uc_email") {
                return models.ErrDuplicateEmail
            }
        }
    }
    return err
}


func (m *UserModel) Authenticate(email, password string) (int, error) {
    // Retrieve the id and hashed password associated with the given email. If no
    // matching email exists, we return the ErrInvalidCredentials error.
    var id int
    var hashedPassword []byte
    row := m.DB.QueryRow("SELECT id, hashed_password FROM users WHERE email = ?", email)
    err := row.Scan(&id, &hashedPassword)
    if err == sql.ErrNoRows {
        return 0, models.ErrInvalidCredentials
    } else if err != nil {
        return 0, err
    }

    // Check whether the hashed password and plain-text password provided match.
    // If they don't, we return the ErrInvalidCredentials error.
    err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
    if err == bcrypt.ErrMismatchedHashAndPassword {
        return 0, models.ErrInvalidCredentials
    } else if err != nil {
        return 0, err
    }

    // Otherwise, the password is correct. Return the user ID.
    return id, nil
}


func (m *UserModel) Get(id int) (*models.User, error) {
    s := &models.User{}

    stmt := `SELECT id, name, email, created, rolesuper, roleadmin, rolevalidator, roletranslator
             FROM users WHERE id = ?`
    err := m.DB.QueryRow(stmt, id).Scan(&s.ID, &s.Name, &s.Email, &s.Created, &s.Super, &s.Admin, &s.Validator,
                                        &s.Translator)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return s, nil
}