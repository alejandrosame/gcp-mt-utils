package files

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    "baliance.com/gooxml/document"
    "baliance.com/gooxml/measurement"
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

func ReadPairsFromTsv(path string) *TranslationPairFile {
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
        filePairs = append(filePairs,
                           models.FilePair{pairs[0], pairs[1],
                                           pairs[2], pairs[3],
                                           pairs[4],
                                           pairs[5], pairs[6]})
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


func ReadPairsFromXlsx(path string) *TranslationPairFile {
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
        filePairs = append(filePairs, models.FilePair{row[0], row[1],
                                                      row[2], row[3],
                                                      row[4],
                                                      row[5], row[6]})
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


func WriteTranslationToDocx(tmp_file, targetText string) string{
    doc := document.New()

    para := doc.AddParagraph()
    run := para.AddRun()

    for _, text := range strings.Split(targetText, "\n") {
        para = doc.AddParagraph()
        para.Properties().SetFirstLineIndent(0.5 * measurement.Inch)
        run = para.AddRun()
        run.AddText(text)
    }

    doc.SaveToFile(tmp_file)

    file, err := os.Open(tmp_file)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    fileStat, _ := file.Stat() //Get info from file
    fileSize := strconv.FormatInt(fileStat.Size(), 10)
    return fileSize
}


func WriteDataset(tmp_file string, pairs []*models.Pair) string{
    file, err := os.Create(tmp_file)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    for _, pair := range pairs {
        _, err := file.WriteString(fmt.Sprintf("%s\t%s\n", pair.SourceText, pair.TargetText))
        if err != nil {
            log.Fatal(err)
        }
    }

    file.Sync()
    fileStat, _ := file.Stat() //Get info from file
    fileSize := strconv.FormatInt(fileStat.Size(), 10)
    return fileSize
}