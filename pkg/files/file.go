package files

import (
    "bufio"
    "log"
    "os"
    "strings"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
)

type TranslationPairFile struct {
    Path            string
    Errors          errors
    Pairs           []*models.FilePair
}


func ReadPairsFromTsv(path string, sourceLanguage string, targetLanguage string) *TranslationPairFile {
    file, err := os.Open(path)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    filePairs := []*models.FilePair{}

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        pairs := strings.Split(scanner.Text(), "\t")

        filePairs = append(filePairs, &models.FilePair{sourceLanguage, targetLanguage, pairs[0], pairs[1]})
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    return &TranslationPairFile{
        path,
        errors(map[string][]string{}),
        filePairs,
    }
}