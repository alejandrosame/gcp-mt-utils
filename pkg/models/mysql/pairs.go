package mysql

import (
    "database/sql"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

type PairModel struct {
    DB *sql.DB
}

func (m *PairModel) Insert(sourceLanguage, targetLanguage, sourceText, targetText string) (int, error) {

	stmt := `INSERT INTO pairs (source_language, target_language, source_text, target_text, created)
    VALUES(?, ?, ?, ?, UTC_TIMESTAMP())`

    result, err := m.DB.Exec(stmt, sourceLanguage, targetLanguage, sourceText, targetText)
    if err != nil {
        return 0, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }

    return int(id), nil
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