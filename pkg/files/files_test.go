package files_test

import (
    "testing"

    "github.com/alejandrosame/gcp-mt-utils/pkg/files"
    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
    "github.com/alejandrosame/gcp-mt-utils/pkg/_testhelpers"
)


func TestReadPairsFromTsv(t *testing.T) {

    expectedPairs := []models.FilePair{
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 1", "a", "b"},
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 2", "c", "d"},
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 3", "e", "f"},
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 4", "g", "h"},
    }

    file := files.ReadPairsFromTsv("./testdata/test_pair.tsv")

    testhelper.Equals(t, expectedPairs, file.Pairs)
}


func TestReadPairsFromXlsx(t *testing.T) {

    expectedPairs := []models.FilePair{
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 1", "a", "b"},
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 2", "c", "d"},
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 3", "e", "f"},
        models.FilePair{"EN", "English-Version",  "ES", "Version-española", "line 4", "g", "h"},
    }

    file := files.ReadPairsFromXlsx("./testdata/test_pair.xlsx")

    testhelper.Equals(t, expectedPairs, file.Pairs)
}


func TestReadAndWriteTranslatedDocx(t *testing.T) {

    text, characterCount, err := files.ExtractTextToTranslateDocx("./testdata/test_doc.docx")

    testhelper.Ok(t, err)
    testhelper.Equals(t, 12490, characterCount)

    filesize := files.WriteTranslatedTextToDocx(text, "./testdata/test_doc.docx", "./testdata/test_doc_out.docx")

    testhelper.Equals(t, "16183", filesize)
}