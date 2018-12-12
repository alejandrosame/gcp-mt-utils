package mysql

import (
    "strings"

    "database/sql"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

type PairModel struct {
    DB *sql.DB
}

func (m *PairModel) Insert(sourceLanguage, sourceVersion, targetLanguage, targetVersion, detail,
                           sourceText, targetText string) (int, error) {

    sqlStr := `INSERT INTO pairs (source_language, sl_text_source, target_language, tl_text_source, text_detail,
                                  source_text, target_text, validated, created, updated)
    VALUES(?, ?, ?, ?, ?, ?, ?, false, UTC_TIMESTAMP(), UTC_TIMESTAMP())`

    result, err := m.DB.Exec(sqlStr, sourceLanguage, sourceVersion, targetLanguage, targetVersion, detail,
                                     sourceText, targetText)
    if err != nil {
        return 0, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }

    return int(id), nil
}


func (m *PairModel) BulkInsertHelper(inputSqlStr string, vals []interface{}) (int64, error) {
    sqlStr := strings.TrimSuffix(inputSqlStr, ",")

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


func (m *PairModel) BulkInsert(pairs []models.FilePair) (int64, error) {

    startingStr := `INSERT INTO pairs (source_language, target_language, sl_text_source, tl_text_source, text_detail,
                                  source_text, target_text, validated, created, updated) VALUES `
    placeholder_part := "(?, ?, ?, ?, ?, ?, ?, false, UTC_TIMESTAMP(), UTC_TIMESTAMP()),"
    total_count_inserted := int64(0)
    number_placeholders := int64(strings.Count(placeholder_part, "?"))

    sqlStr := startingStr
    total_placeholders := int64(0)
    vals := []interface{}{}

    for _, row := range pairs {
        sqlStr += placeholder_part
        total_placeholders += number_placeholders
        vals = append(vals, row.SourceLanguage, row.TargetLanguage, row.SourceVersion, row.TargetVersion,
                      row.Detail, row.SourceText, row.TargetText)

        // MySQL will fail if placeholder count is bigger than 65535, so we need to chunk the inserts
        if total_placeholders + number_placeholders > 65535 {
            // Make partial insert
            count, err := m.BulkInsertHelper(sqlStr, vals)
            if err != nil {
                return 0, err
            }

            total_count_inserted += count

            // Reset placeholder sql string, placeholder counter and val list
            sqlStr = startingStr
            total_placeholders = int64(0)
            vals = []interface{}{}
        }
    }

    return total_count_inserted, nil
}


func (m *PairModel) Get(id int) (*models.Pair, error) {

    stmt := `SELECT id, source_language, sl_text_source, target_language, tl_text_source, source_text, target_text, 
                    text_detail, comments, validated, gcp_dataset,created, updated 
             FROM pairs
             WHERE id = ?`

    p := &models.Pair{}

    err := m.DB.QueryRow(stmt, id).Scan(&p.ID,
                                        &p.SourceLanguage, &p.SourceVersion,
                                        &p.TargetLanguage, &p.TargetVersion,
                                        &p.SourceText, &p.TargetText,
                                        &p.Detail, &p.Comments, &p.Validated, &p.GcpDataset,
                                        &p.Created, &p.Updated)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return p, nil
}


func (m *PairModel) Latest() ([]*models.Pair, error) {

    stmt := `SELECT id, source_language, target_language, source_text, target_text, created FROM pairs
    ORDER BY created DESC, id DESC LIMIT 10`

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


func (m *PairModel) GetNewIDToValidate(sourceLanguage, targetLanguage string) (int, error) {

    stmt := `SELECT id FROM pairs WHERE source_language = ? AND target_language = ? AND NOT validated
    ORDER BY RAND()
    LIMIT 1;`

    p := &models.Pair{}

    err := m.DB.QueryRow(stmt, sourceLanguage, targetLanguage).Scan(&p.ID,)
    if err == sql.ErrNoRows {
        return 0, models.ErrNoRecord
    } else if err != nil {
        return 0, err
    }

    return p.ID, nil
}


func (m *PairModel) GetToValidateFromID(id int) (*models.Pair, error) {

    stmt := "SELECT source_language, target_language FROM pairs WHERE id = ?"

    previous := &models.Pair{}

    err := m.DB.QueryRow(stmt, id).Scan(&previous.SourceLanguage, &previous.TargetLanguage)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    stmt = `SELECT id, source_language, sl_text_source, target_language, tl_text_source, source_text, target_text, 
                   text_detail, comments, validated, gcp_dataset, created, updated
    FROM pairs
    WHERE source_language = ? AND target_language = ? AND NOT validated
    ORDER BY RAND()
    LIMIT 1`

    p := &models.Pair{}

    err = m.DB.QueryRow(stmt, previous.SourceLanguage,
                        previous.TargetLanguage).Scan(&p.ID,
                                                      &p.SourceLanguage, &p.SourceVersion,
                                                      &p.TargetLanguage, &p.TargetVersion,
                                                      &p.SourceText, &p.TargetText,
                                                      &p.Detail, &p.Comments, &p.Validated, &p.GcpDataset,
                                                      &p.Created, &p.Updated)
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

    // TODO

    return nil
}


func (m *PairModel) ValidationStatistics(id int) (*models.ValidationStats, error) {

    sqlStr := `WITH p AS (
                SELECT id, source_language AS sl, target_language AS tl
                FROM pairs
                WHERE id = ?
            ),
            scope_validated AS (
                SELECT COUNT(pairs.id) AS count
                FROM pairs, p
                WHERE pairs.source_language = p.sl AND pairs.target_language = p.tl AND validated = true
            ),
            scope_not_validated AS (
                SELECT COUNT(pairs.id) AS count
                FROM pairs, p
                WHERE pairs.source_language = p.sl AND pairs.target_language = p.tl AND validated = false
            ) SELECT sv.count AS validated, snv.count AS not_validated
              FROM scope_validated AS sv, scope_not_validated AS snv;`

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return nil, err
    }

    stats := &models.ValidationStats{}

    err = stmt.QueryRow(id).Scan(&stats.Validated, &stats.NotValidated)
    if err != nil {
        return nil, err
    }

    stats.Total = stats.Validated + stats.NotValidated
    stats.Percent = 100*float64(stats.Validated)/float64(stats.Total)

    return stats, nil
}