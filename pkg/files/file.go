package files

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    "baliance.com/gooxml/color"
    "baliance.com/gooxml/document"
    "baliance.com/gooxml/measurement"
    "github.com/mholt/archiver"
    "github.com/360EntSecGroup-Skylar/excelize"
)

type TranslationPairFile struct {
    Path            string
    Errors          errors
    Pairs           []models.FilePair
}

type Paragraph struct {
    Runs      []TextRun
}

type TextRun struct {
    Text                string
    TranslationError    bool
}

type TextStruct struct {
    Paragraphs      []Paragraph
    CharacterCount  int
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


func WriteDocxWithoutFormat(tmp_file string, text *TextStruct) string{
    doc := document.New()

    counter := 1
    for _, p := range text.Paragraphs {
        para := doc.AddParagraph()
        para.Properties().SetFirstLineIndent(0.5 * measurement.Inch)
        firstRun := true
        for _, r := range p.Runs {
            // Add paragraph counter
            if firstRun && r.Text != "" {
                run := para.AddRun()
                run.Properties().SetBold(true)
                run.AddText(fmt.Sprintf("%d - ", counter))
                counter = counter + 1
                firstRun = false
            }
            run := para.AddRun()
            if r.TranslationError {
                run.Properties().SetColor(color.Red)
            }else {
                run.Properties()
            }
            run.AddText(r.Text)
        }
    }

    doc.SaveToFile(tmp_file)

    return GetFileSize(tmp_file)
}


func WriteDocxWithFormat(translation *TextStruct, format_file, output_tmp_file string) string {

    doc, err := document.Open(format_file)
    if err != nil {
        return fmt.Sprintf("ERROR OPENING INPUT FILE: %v", err)
    }

    counterP := 1
    for _, p := range doc.Paragraphs() {
        counterR := 1
        for _, r := range p.Runs() {
            text := r.Text()

            if text != "" {
                translatedRun := translation.Paragraphs[counterP-1].Runs[counterR-1]
                r.ClearContent()
                if translatedRun.TranslationError{
                    r.Properties().SetColor(color.Red)
                }
                r.AddText(translatedRun.Text)
            }
            counterR = counterR + 1
        }

        counterP = counterP + 1
    }

    doc.SaveToFile(output_tmp_file)

    return GetFileSize(output_tmp_file)
}


func WriteTranslationInterleavedToDocx(tmp_file string, sourceText, targetText *TextStruct) string{
    doc := document.New()

    para := doc.AddParagraph()
    run := para.AddRun()

    counter := 1
    for idxParagraph, p := range sourceText.Paragraphs {
        paraSource := doc.AddParagraph()
        paraSource.Properties().SetFirstLineIndent(0.5 * measurement.Inch)

        paraTarget := doc.AddParagraph()
        paraTarget.Properties().SetFirstLineIndent(0.5 * measurement.Inch)

        para := doc.AddParagraph()
        para.Properties().SetFirstLineIndent(0.5 * measurement.Inch)

        firstRun := true
        for idxRun, r := range p.Runs {
            // Add paragraph counter
            if firstRun && r.Text != "" {
                run = paraSource.AddRun()
                run.Properties().SetBold(true)
                run.Properties().SetColor(color.Orange)
                run.AddText(fmt.Sprintf("%d - ",counter))

                run = paraTarget.AddRun()
                run.Properties().SetBold(true)
                run.AddText(fmt.Sprintf("%d - ",counter))
                counter = counter + 1

                firstRun = false
            }
            run := para.AddRun()

            run = paraSource.AddRun()
            run.Properties().SetColor(color.Orange)
            run.AddText(r.Text)

            run = paraTarget.AddRun()
            run.Properties()

            targetRun := sourceText.Paragraphs[idxParagraph].Runs[idxRun]

            if targetRun.TranslationError{
                run.Properties().SetColor(color.Red)
            }else{
                run.Properties()
            }
            run.AddText(targetRun.Text)
        }
    }

    doc.SaveToFile(tmp_file)

    return GetFileSize(tmp_file)
}


func ExtractTextToTranslateDocx(input_tmp_file string) (*TextStruct, error) {
    text := TextStruct{}

    doc, err := document.Open(input_tmp_file)
    if err != nil {
        return nil, err
    }

    for _, p := range doc.Paragraphs() {
        var paragraph Paragraph
        for _, r := range p.Runs() {
            textRun := TextRun{}
            currentText := r.Text()
            text.CharacterCount = text.CharacterCount + len([]rune(strings.Replace(currentText, "\n", "", -1)))
            
            textRun.Text = currentText
            textRun.TranslationError = false
            paragraph.Runs = append(paragraph.Runs, textRun)
        }

        text.Paragraphs = append(text.Paragraphs, paragraph)
    }

    return &text, nil
}


func StringToLines(s string) (lines []string, err error) {
    scanner := bufio.NewScanner(strings.NewReader(s))
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    err = scanner.Err()
    return
}


func ConvertPlainTextToTextStruct(plainText string) *TextStruct {
    text := TextStruct{}

    lines, err := StringToLines(plainText)
    if err != nil {
        return &text
    }

    for _, l := range lines {
        characterCount := len([]rune(strings.Replace(l, "\n", "", -1)))
        runs := []TextRun{
            TextRun{l, false},
        }
        p := Paragraph{Runs: runs}

        text.Paragraphs = append(text.Paragraphs, p)
        text.CharacterCount = text.CharacterCount + characterCount
    }

    return &text
}


func ConvertTextStructToPlainText(text *TextStruct) (*string, bool) {
    totalText := ""
    ok := true
    for _, p := range text.Paragraphs {
        runText := ""
        for _, r := range p.Runs {
            runText = runText + r.Text
            if r.TranslationError{
                ok = false
            }
        }
        totalText = totalText + runText + "\n"
    }

    return &totalText, ok
}


func GetFileSize(fileName string) string {
    file, err := os.Open(fileName)
    if err != nil {
        return "ERROR OPENING FILE"
    }
    defer file.Close()

    fileStat, _ := file.Stat() //Get info from file
    return strconv.FormatInt(fileStat.Size(), 10)
}


func WriteDataset(tmp_file string, pairs []*models.Pair) string{
    file, err := os.Create(tmp_file)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    for _, pair := range pairs {
        cleanSourceText := strings.Replace(strings.Replace(strings.Replace(strings.TrimSpace(pair.SourceText),
                                                                           "\t", " ", -1),
                                                           "\r\n", " ", -1),
                                           "\n", " ", -1)
        cleanTargetText := strings.Replace(strings.Replace(strings.Replace(strings.TrimSpace(pair.TargetText),
                                                                            "\t", " ", -1),
                                                           "\r\n", " ", -1),
                                           "\n", " ", -1)
        _, err := file.WriteString(fmt.Sprintf("%s\t%s\n", cleanSourceText, cleanTargetText))
        if err != nil {
            log.Fatal(err)
        }
    }

    file.Sync()
    fileStat, _ := file.Stat() //Get info from file
    fileSize := strconv.FormatInt(fileStat.Size(), 10)
    return fileSize
}


func ArchiveFiles(fileName string, files []string) error {
    return archiver.Archive(files, fileName)
}