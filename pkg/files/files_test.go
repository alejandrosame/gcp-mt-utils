package files_test

import (
    "testing"

    "github.com/alejandrosame/gcp-mt-utils/pkg/files"
    "github.com/alejandrosame/gcp-mt-utils/pkg/models"
    "github.com/alejandrosame/gcp-mt-utils/pkg/_testhelpers"
)


func TestReadPairsFromTsv(t *testing.T) {

    expectedPairs := []*models.FilePair{
        &models.FilePair{"EN", "ES", "a", "b"},
        &models.FilePair{"EN", "ES", "c", "d"},
        &models.FilePair{"EN", "ES", "e", "f"},
        &models.FilePair{"EN", "ES", "g", "h"},
    }

    file := files.ReadPairsFromTsv("./testdata/test_pair.tsv", "EN", "ES")

    testhelper.Equals(t, expectedPairs, file.Pairs)
}