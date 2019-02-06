package mysql

import (
    "database/sql"
    "fmt"
    "strings"
    "time"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    "github.com/go-sql-driver/mysql"
    "golang.org/x/crypto/bcrypt"
)

type UserModel struct {
    DB *sql.DB
}

func (m *UserModel) Insert(name, email, password, role string) error {
    // Create a bcrypt hash of the plain-text password.
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return err
    }

    var roleStmt string
    if role == "admin" {
        roleStmt = "roleadmin"
    }else if role == "validator" {
        roleStmt = "rolevalidator"
    } else if role == "translator" {
        roleStmt = "roletranslator"
    }

    stmt := fmt.Sprintf(`INSERT INTO users (name, email, hashed_password, %s, created)
    VALUES(?, ?, ?, true, UTC_TIMESTAMP())`, roleStmt)

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


func (m *UserModel) GetUserName(email string) (string, error) {
    u := &models.UserLimit{}

    stmt := `SELECT name FROM users
            WHERE email = ?;`
    err := m.DB.QueryRow(stmt, email).Scan(&u.Name)
    if err == sql.ErrNoRows {
        return "", models.ErrNoRecord
    } else if err != nil {
        return "", err
    }

    return u.Name, nil
}


func (m *UserModel) UpdatePassword(email, password string) (int, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return 0, err
    }

    sqlStr := `UPDATE users
               SET hashed_password = ?
               WHERE email = ?`

    _, err = m.DB.Exec(sqlStr, hashedPassword, email)
    if err != nil {
        return 0, err
    }

    return 1, nil
}


func (m *UserModel) Exists(email string) (bool, error) {
    s := &models.User{}

    stmt := `SELECT id, name, email, created, rolesuper, roleadmin, rolevalidator, roletranslator
             FROM users WHERE email = ?`
    err := m.DB.QueryRow(stmt, email).Scan(&s.ID, &s.Name, &s.Email, &s.Created, &s.Super, &s.Admin, &s.Validator,
                                           &s.Translator)
    if err == sql.ErrNoRows {
        return false, nil
    } else if err != nil {
        return false, err
    }

    return true, nil
}


func (m *UserModel) GetRoleLimit(role string) (*models.RoleLimit, error) {
    r := &models.RoleLimit{}

    stmt := `SELECT user_role, character_limit
             FROM role_character_limit WHERE user_role = ?`
    err := m.DB.QueryRow(stmt, role).Scan(&r.UserRole, &r.CharacterLimit)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return r, nil
}

func (m *UserModel) UpdateRoleLimit(role string, limit int) (string, error) {
    sqlStr := `UPDATE role_character_limit
               SET character_limit = ?
               WHERE user_role = ?`

    _, err := m.DB.Exec(sqlStr, limit, role)
    if err != nil {
        return "", err
    }

    return role, nil
}


func (m *UserModel) GetUserLimit(id int) (*models.UserLimit, error) {
    u := &models.UserLimit{}

    stmt := `SELECT u.id, u.rolesuper, u.roleadmin, u.rolevalidator, u.roletranslator, u.name, u.email,
                   COALESCE(ucl.character_limit, 0) AS character_limit,
                   COALESCE(ucl.character_limit, 0) +
                   (SELECT character_limit FROM role_character_limit WHERE user_role = 'all') AS total_character_limit,
                   COALESCE(consumed.total_characters_translated, 0) AS total_characters_translated
            FROM
                users u
            LEFT JOIN
                user_character_limit ucl ON u.id = ucl.user_id
            LEFT JOIN
                (SELECT user_id, SUM(characters_translated) AS total_characters_translated
                 FROM user_characters_consumed
                 WHERE month(date) = month(UTC_TIMESTAMP()) AND year(date) = year(UTC_TIMESTAMP())
                 GROUP BY user_id, month(date), year(date)
                ) consumed
                ON u.id = consumed.user_id
            WHERE u.id = ?;`
    err := m.DB.QueryRow(stmt, id).Scan(&u.ID,
                                        &u.Super, &u.Admin, &u.Validator, &u.Translator,
                                        &u.Name, &u.Email,
                                        &u.CharacterLimit, &u.TotalLimit, &u.TotalTranslated)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return u, nil
}

func (m *UserModel) GetAllUserLimits() ([]*models.UserLimit, error) {

    stmt := `SELECT u.id, u.rolesuper, u.roleadmin, u.rolevalidator, u.roletranslator, u.name, u.email,
                   COALESCE(ucl.character_limit, 0) AS character_limit,
                   COALESCE(ucl.character_limit, 0) +
                   (SELECT character_limit FROM role_character_limit WHERE user_role = 'all') AS total_character_limit,
                   COALESCE(consumed.total_characters_translated, 0) AS total_characters_translated
            FROM
                users u
            LEFT JOIN
                user_character_limit ucl ON u.id = ucl.user_id
            LEFT JOIN
                (SELECT user_id, SUM(characters_translated) AS total_characters_translated
                 FROM user_characters_consumed
                 WHERE month(date) = month(UTC_TIMESTAMP()) AND year(date) = year(UTC_TIMESTAMP())
                 GROUP BY user_id, month(date), year(date)
                ) consumed
                ON u.id = consumed.user_id;`

    rows, err := m.DB.Query(stmt)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    limits := []*models.UserLimit{}

    for rows.Next() {
        u := &models.UserLimit{}

        err = rows.Scan(&u.ID,
                        &u.Super, &u.Admin, &u.Validator, &u.Translator,
                        &u.Name, &u.Email,
                        &u.CharacterLimit, &u.TotalLimit, &u.TotalTranslated)
        if err != nil {
            return nil, err
        }
        limits = append(limits, u)
    }

    if err = rows.Err(); err != nil {
        return nil, err
    }

    return limits, nil
}

func (m *UserModel) UpdateUserLimit(id int, limit int) (int, error) {
    sqlStr := `UPDATE user_character_limit
               SET character_limit = ?
               WHERE user_id = ?`

    result, err := m.DB.Exec(sqlStr, limit, id)
    if err != nil {
        return 0, err
    }

    count, err := result.RowsAffected()
    if err != nil {
        return 0, err
    }

    // User still does not have a translation limit defined, we need to insert it instead of updating it
    if count == 0 {
        sqlStr := `INSERT into user_character_limit (character_limit, user_id)
                   VALUES (?, ?)`

        _, err := m.DB.Exec(sqlStr, limit, id)
        if err != nil {
            return 0, err
        }
    }

    return id, nil
}


func (m *UserModel) UpdateUserCharactersConsumed(id int, characterCount int) (int, error) {
    sqlStr := `INSERT into user_characters_consumed (user_id, characters_translated, date)
               VALUES (?, ?, UTC_TIMESTAMP())`

    _, err := m.DB.Exec(sqlStr, id, characterCount)
    if err != nil {
        return 0, err
    }

    return id, nil
}


func (m *UserModel) InsertPasswordChangeRequest(email string) (*models.PasswordChangeRequest, error) {
    stmt := `SELECT count(*) FROM users WHERE email = ?`

    var count int
    row := m.DB.QueryRow(stmt, email)
    err := row.Scan(&count)
    if err != nil {
        return nil, err
    }
    if count != 1{
        return nil, models.ErrEmailNotFound
    }

    stmt = `SELECT count(*) FROM change_password_token WHERE email = ? AND UTC_TIMESTAMP() <= expires AND NOT used`

    row = m.DB.QueryRow(stmt, email)
    err = row.Scan(&count)
    if err != nil {
        return nil, err
    }
    if count != 0{
        return nil, models.ErrTokenStillValid
    }

    seed := strings.Join([]string{email, time.Now().String()}, "")

    token, err := bcrypt.GenerateFromPassword([]byte(seed), 12)
    if err != nil {
        return nil, err
    }

    stmt = `INSERT INTO change_password_token (email, token, created, expires)
    VALUES(?, ?, UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL 10 MINUTE))`

    result, err := m.DB.Exec(stmt, email, token)
    if err != nil {
        return nil, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return nil, err
    }

    return m.GetPasswordChangeToken(int(id))
}


func (m *UserModel) CheckPasswordChangeToken(email, token string) (*models.PasswordChangeRequest, error) {
    stmt := `SELECT id FROM change_password_token
    WHERE token = ? AND email= ? AND UTC_TIMESTAMP() <= expires AND NOT used`

    var id int
    row := m.DB.QueryRow(stmt, token, email)
    err := row.Scan(&id)
    if err == sql.ErrNoRows {
        return nil, models.ErrTokenNotFound
    } else if err != nil {
        return nil, err
    }
    return m.GetPasswordChangeToken(int(id))
}


func (m *UserModel) PasswordChangeTokenExists(token string) (bool, error) {
    stmt := `SELECT id FROM change_password_token
    WHERE token = ? AND UTC_TIMESTAMP() <= expires AND NOT used`

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


func (m *UserModel) MarkPasswordChangeTokenAsUsed(token string) (int, error) {
    stmt := `UPDATE change_password_token SET used = true
    WHERE token = ?`

    result, err := m.DB.Exec(stmt, token)
    if err != nil {
        return 0, err
    }

    count, err := result.RowsAffected()
    if err != nil {
        return 0, err
    }

    // User still does not have a translation limit defined, we need to insert it instead of updating it
    if count != 1 {
        return 0, models.ErrTokenNotFound
    }

    return 1, nil

}


func (m *UserModel) GetPasswordChangeToken(id int) (*models.PasswordChangeRequest, error) {
    p := &models.PasswordChangeRequest{}

    stmt := `SELECT id, email, token, created, expires, used
             FROM change_password_token WHERE id = ?`
    row := m.DB.QueryRow(stmt, id)
    err := row.Scan(&p.ID, &p.Email, &p.Token, &p.Created, &p.Expires, &p.Used)
    if err == sql.ErrNoRows {
        return nil, models.ErrTokenNotFound
    } else if err != nil {
        return nil, err
    }

    return p, nil
}