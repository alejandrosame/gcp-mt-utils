package mysql

import (
    "fmt"
    "log"
    "regexp"
    "strconv"
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


func (m *PairModel) Edit(id int, sourceText, targetText, comments string) (int, error) {
    sqlStr := `UPDATE pairs SET sl_text_source = ?, tl_text_source = ?, comments = ?, validated = false,
                          updated = UTC_TIMESTAMP()
               WHERE id = ?`

    _, err := m.DB.Exec(sqlStr, sourceText, targetText, comments, id)
    if err != nil {
        return 0, err
    }

    return id, nil
}


func (m *PairModel) EditComments(id int, comments string) (int, error) {
    sqlStr := `UPDATE pairs SET comments = ?, updated = UTC_TIMESTAMP()
               WHERE id = ?`

    _, err := m.DB.Exec(sqlStr, comments, id)
    if err != nil {
        return 0, err
    }

    return id, nil
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


func (m *PairModel) GetBibleBooks() ([]*models.BibleBook, error) {

    stmt := `SELECT id, name, chapter, testament
             FROM bible_structure
             ORDER BY id ASC`

    rows, err := m.DB.Query(stmt)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    books := []*models.BibleBook{}

    for rows.Next() {
        b := &models.BibleBook{}

        err = rows.Scan(&b.ID,
                        &b.Name, &b.Chapter, &b.Testament)
        if err != nil {
            return nil, err
        }
        books = append(books, b)
    }

    if err = rows.Err(); err != nil {
        return nil, err
    }

    return books, nil
}


func (m *PairModel) GetBook(id int) (*models.BibleBook, error) {

    stmt := `SELECT id, name, chapter, testament
             FROM bible_structure
             WHERE id = ?`

    b := &models.BibleBook{}
    err := m.DB.QueryRow(stmt, id).Scan(&b.ID,
                        &b.Name, &b.Chapter, &b.Testament)
    if err == sql.ErrNoRows {
        return nil, models.ErrNoRecord
    } else if err != nil {
        return nil, err
    }

    return b, nil
}


func (m *PairModel) GetBookFromDetail(detail string) (*models.BibleBook, error) {
    detailRe := regexp.MustCompile("book (\\d+), chapter(\\d+), verse (\\d+)")
    match := detailRe.FindStringSubmatch(detail)
    bookId, _ := strconv.Atoi(match[1])
    chapterId, _ := strconv.Atoi(match[2])

    b, err := m.GetBook(bookId)
    if err != nil {
        return nil, err
    }

    b.Chapter = chapterId

    return b, nil
}


func (m *PairModel) GetPairs(infoLog *log.Logger, sourceLanguage, targetLanguage string, book, chapter int) ([]*models.Pair, error) {

    stmt := `SELECT id, source_language, sl_text_source, target_language, tl_text_source, source_text, target_text,
                    text_detail, comments, validated, gcp_dataset, created, updated
             FROM pairs
             WHERE source_language = ? AND target_language = ? AND text_detail LIKE ?
             ORDER BY id ASC`


    rows, err := m.DB.Query(stmt, sourceLanguage, targetLanguage,
                                  fmt.Sprintf("book %d, chapter%d,%s", book, chapter, "%"))
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


func (m *PairModel) GetNewIDToValidate(sourceLanguage, targetLanguage string, book, chapter int) (int, error) {

    stmt := `SELECT id FROM pairs
    WHERE source_language = ? AND target_language = ? AND text_detail LIKE ? AND NOT validated
    ORDER BY RAND()
    LIMIT 1;`

    p := &models.Pair{}

    err := m.DB.QueryRow(stmt, sourceLanguage, targetLanguage,
                               fmt.Sprintf("book %d, chapter%d,%s", book, chapter, "%")).Scan(&p.ID,)
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


func (m *PairModel) ValidationStatisticsBookChapter(id int) (*models.ValidationStats, error) {

    p, err := m.Get(id)
    if err != nil {
        return nil, err
    }

    b, err := m.GetBookFromDetail(p.Detail)
    if err != nil {
        return nil, err
    }

    sqlStr := `WITH p AS (
                SELECT id, source_language AS sl, target_language AS tl
                FROM pairs
                WHERE id = ?
            ),
            scope_validated AS (
                SELECT COUNT(pairs.id) AS count
                FROM pairs, p
                WHERE pairs.source_language = p.sl AND pairs.target_language = p.tl AND pairs.text_detail LIKE ? AND validated = true
            ),
            scope_not_validated AS (
                SELECT COUNT(pairs.id) AS count
                FROM pairs, p
                WHERE pairs.source_language = p.sl AND pairs.target_language = p.tl AND pairs.text_detail LIKE ? AND validated = false
            ) SELECT sv.count AS validated, snv.count AS not_validated
              FROM scope_validated AS sv, scope_not_validated AS snv;`

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return nil, err
    }

    stats := &models.ValidationStats{}

    detailLike := fmt.Sprintf("book %d, chapter%d,%s", b.ID, b.Chapter, "%")

    err = stmt.QueryRow(id, detailLike, detailLike).Scan(&stats.Validated, &stats.NotValidated)
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
              WHERE source_language = ? AND target_language = ? AND gcp_dataset IS NULL AND validated = true
              ORDER BY id ASC`

    rows, err := m.DB.Query(sqlStr, sourceLanguage, targetLanguage)
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


func (m *PairModel) GetValidatedNotExportedFromChapter(sourceLanguage, targetLanguage string,
                                                       book, chapter int) ([]*models.Pair, error) {
    sqlStr := `SELECT id, source_language, sl_text_source, target_language, tl_text_source,
                     source_text, target_text, text_detail, comments, validated,
                     gcp_dataset,created, updated
              FROM pairs
              WHERE source_language = ? AND target_language = ? AND text_detail LIKE ? AND
                    gcp_dataset IS NULL AND validated = true
              ORDER BY id ASC`

    detailLike := fmt.Sprintf("book %d, chapter%d,%s", book, chapter, "%")

    rows, err := m.DB.Query(sqlStr, sourceLanguage, targetLanguage, detailLike)
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


func (m *PairModel) ValidateAllPairsFromChapter(sourceLanguage, targetLanguage string, book, chapter int) (error) {
    sqlStr := `UPDATE pairs SET validated = true
    WHERE source_language = ? AND target_language = ? AND text_detail LIKE ? AND
          validated = false`

    detailLike := fmt.Sprintf("book %d, chapter%d,%s", book, chapter, "%")
    _, err := m.DB.Exec(sqlStr, sourceLanguage, targetLanguage, detailLike)
    if err != nil {
        return err
    }

    return nil
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