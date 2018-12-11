package mysql

import (
    "strings"

    "database/sql"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

type PairModel struct {
    DB *sql.DB
}

func (m *PairModel) Insert(sourceLanguage, targetLanguage, sourceText, targetText string) (int, error) {

    sqlStr := `INSERT INTO pairs (source_language, target_language, source_text, target_text, created)
    VALUES(?, ?, ?, ?, UTC_TIMESTAMP())`

    result, err := m.DB.Exec(sqlStr, sourceLanguage, targetLanguage, sourceText, targetText)
    if err != nil {
        return 0, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }

    return int(id), nil
}


func (m *PairModel) BulkInsert(pairs []models.FilePair) (int64, error) {

    sqlStr := "INSERT INTO pairs (source_language, target_language, source_text, target_text, created) VALUES "
    vals := []interface{}{}

    for _, row := range pairs {
        sqlStr += "(?, ?, ?, ?, UTC_TIMESTAMP()),"
        vals = append(vals, row.SourceLanguage, row.TargetLanguage, row.SourceText, row.TargetText)
    }

    sqlStr = strings.TrimSuffix(sqlStr, ",")

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return 0, err
    }

    result, err := stmt.Exec(vals...)
    if err != nil {
        return 0, err
    }

    count, err := result.RowsAffected()
    if err != nil {
        return 0, err
    }

    return count, nil
}


func (m *PairModel) Get(id int) (*models.Pair, error) {

    stmt := `SELECT id, source_language, target_language, source_text, target_text, created FROM pairs
    WHERE id = ?`

    p := &models.Pair{}

    err := m.DB.QueryRow(stmt, id).Scan(&p.ID, &p.SourceLanguage, &p.TargetLanguage, &p.SourceText, &p.TargetText, 
                                        &p.Created)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return p, nil
}


func (m *PairModel) Latest() ([]*models.Pair, error) {

    stmt := `SELECT id, source_language, target_language, source_text, target_text, created FROM pairs
    ORDER BY created DESC LIMIT 10`

    rows, err := m.DB.Query(stmt)
    if err != nil {
        return nil, err
    }

    defer rows.Close()

    pairs := []*models.Pair{}

    for rows.Next() {
        p := &models.Pair{}

        err = rows.Scan(&p.ID, &p.SourceLanguage, &p.TargetLanguage, &p.SourceText, &p.TargetText, &p.Created)
        if err != nil {
            return nil, err
        }
        pairs = append(pairs, p)
    }

    if err = rows.Err(); err != nil {
        return nil, err
    }

    return pairs, nil
}


func (m *PairModel) GetToValidate(sourceLanguage, targetLanguage string) (*models.Pair, error) {

    stmt := `SELECT id, source_language, target_language, source_text, target_text, created FROM pairs
    WHERE source_language = ? AND target_language = ? AND NOT validated`

    p := &models.Pair{}

    err := m.DB.QueryRow(stmt, sourceLanguage, 
                         targetLanguage).Scan(&p.ID, &p.SourceLanguage, &p.TargetLanguage, &p.SourceText, &p.TargetText, 
                                             &p.Created)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return p, nil
}


func (m *PairModel) GetToValidateFromID(id int) (*models.Pair, error) {

    stmt := `SELECT id, source_language, target_language, source_text, target_text, created FROM pairs
    WHERE id = ?`

    p := &models.Pair{}

    err := m.DB.QueryRow(stmt, id).Scan(&p.ID, &p.SourceLanguage, &p.TargetLanguage, &p.SourceText, &p.TargetText, 
                                        &p.Created)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    sourceLanguage := p.SourceLanguage
    targetLanguage := p.TargetLanguage

    stmt = `SELECT id, source_language, target_language, source_text, target_text, created FROM pairs
    WHERE source_language = ? AND target_language = ? AND NOT validated`

    p = &models.Pair{}

    err = m.DB.QueryRow(stmt, sourceLanguage, 
                         targetLanguage).Scan(&p.ID, &p.SourceLanguage, &p.TargetLanguage, &p.SourceText, &p.TargetText, 
                                             &p.Created)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return p, nil
}


func (m *PairModel) Validate(id int) error {
    
    sqlStr := `UPDATE pairs SET validated = true WHERE id = ?`

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return err
    }

    _, err = stmt.Exec(id)
    if err != nil {
        return err
    }
    
    return err
}


func (m *PairModel) Update(id int) error {
    /*
    stmt := `SELECT id, source_language, target_language, source_text, target_text, created FROM pairs
    WHERE source_language = ? AND target_language = ?`

    p := &models.Pair{}

    err := m.DB.QueryRow(stmt, sourceLanguage, 
                         targetLanguage).Scan(&p.ID, &p.SourceLanguage, &p.TargetLanguage, &p.SourceText, &p.TargetText, 
                                             &p.Created)
    if err != nil {
        return err
    }
    */
    return nil
}