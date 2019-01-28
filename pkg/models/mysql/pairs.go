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
    sqlStr := `UPDATE pairs SET source_text = ?, target_text = ?, comments = ?, validated = false,
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


func (m *PairModel) GetBibleBooks(sourceLanguage, targetLanguage string) ([]*models.BibleBook, error) {

    sqlStr := `SELECT id, name, chapter, testament
               FROM bible_structure
               ORDER BY id ASC`

    rows, err := m.DB.Query(sqlStr)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    allStats, err := m.ValidationStatisticsAllBooksChapters(sourceLanguage, targetLanguage)
    if err != nil {
        return nil, err
    }
    books := []*models.BibleBook{}

    bookIdx := 1
    for rows.Next() {
        b := &models.BibleBook{}
        chapterStats := []*models.ValidationStats{}
        bookStats := models.ValidationStats{Validated: 0, NotValidated: 0, Total: 0, Percent: 0}

        err = rows.Scan(&b.ID,
                        &b.Name, &b.Chapter, &b.Testament)
        if err != nil {
            return nil, err
        }

        for chapterIdx := 1; chapterIdx <= b.Chapter; chapterIdx++ {
            stats := models.ValidationStats{Validated: 0, NotValidated: 0, Total: 0, Percent: 100}

            _, exists := (*allStats)[bookIdx]
            if exists {
                _, exists := (*allStats)[bookIdx][chapterIdx]
                if exists {
                    stats = (*allStats)[bookIdx][chapterIdx]
                    bookStats.Validated += stats.Validated
                    bookStats.NotValidated += stats.NotValidated
                    bookStats.Total += stats.Total
                }
            }

            chapterStats = append(chapterStats, &stats)
        }

        if bookStats.Total == 0 {
            bookStats.Percent = 100
        } else {
            bookStats.Percent = 100*float64(bookStats.Validated)/float64(bookStats.Total)
        }

        b.Stats = &bookStats
        b.ChapterStats = chapterStats
        books = append(books, b)
        bookIdx++
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


func (m *PairModel) Unvalidate(id int) error {

    sqlStr := `UPDATE pairs SET validated = false WHERE id = ?`

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


func (m *PairModel) ValidationStatisticsAllBooksChapters(sourceLanguage, targetLanguage string) (*map[int]map[int]models.ValidationStats, error) {

    bookStats := make(map[int]map[int]models.ValidationStats)
    var stats models.ValidationStats
    var bookIdx int
    var chapterIdx int

    sqlStr := `SELECT book,
               chapter,
               COUNT(CASE WHEN validated=true THEN 1 END) as validated,
               COUNT(CASE WHEN validated=false THEN 1 END) as not_validated,
               COUNT(CASE WHEN true THEN 1 END) as total
        FROM (SELECT CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(text_detail, ',', 1), " ", -1) AS UNSIGNED) AS book,
                     CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(text_detail, ',', 2), "r", -1) AS UNSIGNED) AS chapter,
                     source_language,
                     target_language,
                    validated FROM pairs
              WHERE source_language = ? AND
                    target_language = ?
             ) AS expanded_pairs
        GROUP BY source_language,
                 target_language,
                 book,
                 chapter`


    rows, err := m.DB.Query(sqlStr, sourceLanguage, targetLanguage)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        stats = models.ValidationStats{}

        err := rows.Scan(&bookIdx, &chapterIdx, &stats.Validated, &stats.NotValidated, &stats.Total)
        if err != nil {
            return nil, err
        }
        stats.Percent = 100*float64(stats.Validated)/float64(stats.Total)

        _, exists := bookStats[bookIdx]
        if !exists {
            bookStats[bookIdx] = make(map[int]models.ValidationStats)
        }

        bookStats[bookIdx][chapterIdx] = stats
    }

    return &bookStats, nil
}


func (m *PairModel) ValidationStatistics(sourceLanguage, targetLanguage string) (*models.ValidationStats, error) {

    sqlStr := `SELECT COUNT(CASE WHEN validated=true THEN 1 END) as validated,
                      COUNT(CASE WHEN validated=false THEN 1 END) as not_validated,
                      COUNT(CASE WHEN true THEN 1 END) as total
                FROM pairs
                WHERE source_language = ? AND target_language = ?`

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return nil, err
    }

    return m.ExecuteValidationStatistics(stmt.QueryRow(sourceLanguage, targetLanguage))
}


func (m *PairModel) ValidationStatisticsBook(sourceLanguage, targetLanguage string, book int) (*models.ValidationStats, error) {

    sqlStr := `SELECT COUNT(CASE WHEN validated=true THEN 1 END) as validated,
                      COUNT(CASE WHEN validated=false THEN 1 END) as not_validated,
                      COUNT(CASE WHEN true THEN 1 END) as total
                FROM (SELECT CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(text_detail, ',', 1), " ", -1) AS UNSIGNED) AS book,
                             source_language,
                             target_language,
                            validated FROM pairs
                      WHERE source_language = ? AND
                            target_language = ?
                     ) AS expanded_pairs
                WHERE book = ?
                GROUP BY source_language,
                         target_language,
                         book,
                         chapter`

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return nil, err
    }

    return m.ExecuteValidationStatistics(stmt.QueryRow(sourceLanguage, targetLanguage, book))
}

func (m *PairModel) ValidationStatisticsBookChapter(sourceLanguage, targetLanguage, detail string) (*models.ValidationStats, error) {

    b, err := m.GetBookFromDetail(detail)
    if err != nil {
        return nil, err
    }

    sqlStr := `SELECT COUNT(CASE WHEN validated=true THEN 1 END) as validated,
                      COUNT(CASE WHEN validated=false THEN 1 END) as not_validated,
                      COUNT(CASE WHEN true THEN 1 END) as total
                FROM pairs
                WHERE source_language = ? AND target_language = ? AND text_detail LIKE ?`

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return nil, err
    }

    detailLike := fmt.Sprintf("book %d, chapter%d,%s", b.ID, b.Chapter, "%")

    return m.ExecuteValidationStatistics(stmt.QueryRow(sourceLanguage, targetLanguage, detailLike))
}


func (m *PairModel) ExecuteValidationStatistics(query *sql.Row) (*models.ValidationStats, error) {
    stats := &models.ValidationStats{}

    err := query.Scan(&stats.Validated, &stats.NotValidated, &stats.Total)
    if err != nil {
        return nil, err
    }

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

func (m *PairModel) ValidateSelectedPairsFromChapter(sourceLanguage, targetLanguage string, book, chapter int,
                                                     idList string) (error) {

    detailLike := fmt.Sprintf("book %d, chapter%d,%s", book, chapter, "%")

    re := regexp.MustCompile(`\b\d+\b`)
    substitution := `?`
    idsPlaceholder := re.ReplaceAllString(idList, substitution)

    sqlStr := fmt.Sprintf(`UPDATE pairs SET validated = true
    WHERE source_language = ? AND target_language = ? AND text_detail LIKE ? AND
          id IN (%s) AND validated = false`, idsPlaceholder)

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return err
    }

    t := strings.Split(idList, ",")
    t = append([]string{sourceLanguage, targetLanguage, detailLike}, t...)
    vals := make([]interface{}, len(t))
    for i, v := range t {
        vals[i] = v
    }

    _, err = stmt.Exec(vals...)
    if err != nil {
        return err
    }

    return nil
}


func (m *PairModel) UnvalidateSelectedPairsFromChapter(sourceLanguage, targetLanguage string, book, chapter int,
                                                     idList string) (error) {

    detailLike := fmt.Sprintf("book %d, chapter%d,%s", book, chapter, "%")

    re := regexp.MustCompile(`\b\d+\b`)
    substitution := `?`
    idsPlaceholder := re.ReplaceAllString(idList, substitution)

    sqlStr := fmt.Sprintf(`UPDATE pairs SET validated = false
    WHERE source_language = ? AND target_language = ? AND text_detail LIKE ? AND
          id IN (%s) AND validated = true`, idsPlaceholder)

    stmt, err := m.DB.Prepare(sqlStr)
    if err != nil {
        return err
    }

    t := strings.Split(idList, ",")
    t = append([]string{sourceLanguage, targetLanguage, detailLike}, t...)
    vals := make([]interface{}, len(t))
    for i, v := range t {
        vals[i] = v
    }

    _, err = stmt.Exec(vals...)
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