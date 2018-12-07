package files

import (
    "bufio"
    "log"
    "os"
    "strings"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    "github.com/360EntSecGroup-Skylar/excelize"
)

type TranslationPairFile struct {
    Path            string
    Errors          errors
    Pairs           []models.FilePair
}


func (tpf *TranslationPairFile) Valid() bool {
    return len(tpf.Errors) == 0
}

func ReadPairsFromTsv(path string, sourceLanguage string, targetLanguage string) *TranslationPairFile {
    filePairs := []models.FilePair{}
    errors := errors(map[string][]string{})

    file, err := os.Open(path)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()


    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        pairs := strings.Split(scanner.Text(), "\t")
        filePairs = append(filePairs, models.FilePair{sourceLanguage, targetLanguage, pairs[0], pairs[1]})
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    if len(filePairs) == 0 {
        errors.Add("fileName", "File is empty")
    }

    return &TranslationPairFile{
        path,
        errors,
        filePairs,
    }
}


func ReadPairsFromXlsx(path string, sourceLanguage string, targetLanguage string) *TranslationPairFile {
    filePairs := []models.FilePair{}
    errors := errors(map[string][]string{})

    xlsx, err := excelize.OpenFile(path)
    if err != nil {
        log.Fatal(err)
    }

    // Get all the rows in the first sheet
    sheet_name :=  xlsx.GetSheetMap()[1]
    rows := xlsx.GetRows(sheet_name)
    for _, row := range rows {
        filePairs = append(filePairs, models.FilePair{sourceLanguage, targetLanguage, row[0], row[1]})
    }

    if len(filePairs) == 0 {
        errors.Add("fileName", "File is empty")
    }

    return &TranslationPairFile{
        path,
        errors,
        filePairs,
    }
}