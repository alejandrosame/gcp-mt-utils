package mysql

import (
    "fmt"
    "log"
    "regexp"
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

    // If total_placeholders is not 0, we still need to insert data
    if total_placeholders != 0 {
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


func (m *PairModel) Latest(sourceLanguage, targetLanguage string) ([]*models.Pair, error) {

    stmt := `SELECT id, source_language, sl_text_source, target_language, tl_text_source, source_text, target_text, 
                    text_detail, comments, validated, gcp_dataset, created, updated 
             FROM pairs
             WHERE source_language = ? AND target_language = ?
             ORDER BY created DESC, id DESC LIMIT 10`

    rows, err := m.DB.Query(stmt, sourceLanguage, targetLanguage)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    pairs := []*models.Pair{}

    for rows.Next() {
        p := &models.Pair{}

        err = rows.Scan(&p.ID,
                        &p.SourceLanguage, &p.SourceVersion,
                        &p.TargetLanguage, &p.TargetVersion,
                        &p.SourceText, &p.TargetText,
                        &p.Detail, &p.Comments, &p.Validated, &p.GcpDataset,
                        &p.Created, &p.Updated)
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

    stmt := `SELECT id, source_language, sl_text_source, target_language, tl_text_source, source_text, target_text,
                    text_detail, comments, validated, gcp_dataset, created, updated
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


func (m *PairModel) GetValidatedNotExported(sourceLanguage, targetLanguage string) ([]*models.Pair, error) {
    sqlStr := `SELECT id, source_language, sl_text_source, target_language, tl_text_source,
                     source_text, target_text, text_detail, comments, validated,
                     gcp_dataset,created, updated
              FROM pairs
              WHERE gcp_dataset IS NULL AND validated = true
              ORDER BY id ASC`

    rows, err := m.DB.Query(sqlStr)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    pairs := []*models.Pair{}

    for rows.Next() {
        p := &models.Pair{}

        err = rows.Scan(&p.ID,
                        &p.SourceLanguage, &p.SourceVersion,
                        &p.TargetLanguage, &p.TargetVersion,
                        &p.SourceText, &p.TargetText,
                        &p.Detail, &p.Comments, &p.Validated, &p.GcpDataset,
                        &p.Created, &p.Updated)
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


func (m *PairModel) DatasetIsUsed(datasetName string) (bool, error) {
    sqlStr := ` SELECT count(*) as count
                FROM pairs
                WHERE gcp_dataset = ?
                LIMIT 1`

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return true, err
    }

    count := 0

    err = stmt.QueryRow(datasetName).Scan(&count)
    if err != nil {
        return true, err
    }

    return count == 1, nil
}


func (m *PairModel) GetAndMarkedExported(infoLog, errorLog *log.Logger, idList, datasetName string) ([]*models.Pair, error) {

    b, err := m.DatasetIsUsed(datasetName)
    if err != nil {
        return nil, err
    }

    if b {
        return nil, models.ErrDuplicateDataset
    }

    re := regexp.MustCompile(`\b\d+\b`)
    substitution := `?`
    idsPlaceholder := re.ReplaceAllString(idList, substitution)

    sqlStr := fmt.Sprintf(`UPDATE pairs SET gcp_dataset = ?
             WHERE id IN (%s)`, idsPlaceholder)

    infoLog.Println(sqlStr)

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return nil, err
    }

    t := strings.Split(idList, ",")
    t = append([]string{datasetName}, t...)
    vals := make([]interface{}, len(t))
    for i, v := range t {
        vals[i] = v
    }

    _, err = stmt.Exec(vals...)
    if err != nil {
        return nil, err
    }

    sqlStr = `SELECT id, source_language, sl_text_source, target_language, tl_text_source,
                     source_text, target_text, text_detail, comments, validated,
                     gcp_dataset,created, updated
              FROM pairs
              WHERE gcp_dataset = ?`

    rows, err := m.DB.Query(sqlStr, datasetName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    pairs := []*models.Pair{}

    for rows.Next() {
        p := &models.Pair{}

        err = rows.Scan(&p.ID,
                        &p.SourceLanguage, &p.SourceVersion,
                        &p.TargetLanguage, &p.TargetVersion,
                        &p.SourceText, &p.TargetText,
                        &p.Detail, &p.Comments, &p.Validated, &p.GcpDataset,
                        &p.Created, &p.Updated)
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