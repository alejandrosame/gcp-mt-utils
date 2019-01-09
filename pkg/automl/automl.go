package automl

import (
    "bufio"
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"
    "time"

    "github.com/tidwall/gjson"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

// Types for responses
type Content struct {
    Content string `json:"content"`
}

type TranslatedContent struct {
    TranslatedContent Content `json:"translatedContent"`
}

type Translation struct {
    Translation TranslatedContent `json:"translation"`
}

type TranslationAPIResponse struct {
    PayloadList []Translation `json:"payload"`
}

type TranslationModelMetadata struct {
    BaseModel string `json:"baseModel"`
    SourceLanguageCode string `json:"sourceLanguageCode"`
    TargetLanguageCode string `json:"targetLanguageCode"`
}

type Model struct {
   Name string `json:"name"`
   DisplayName string `json:"displayName"`
   DatasetId string `json:"datasetId"`
   CreateTime time.Time `json:"createTime"`
   UpdateTime time.Time `json:"updateTime"`
   DeploymentState string `json:"deploymentState"`
   TranslationModelMetadata TranslationModelMetadata `json:"translationModelMetadata"`
}

type ListModelAPIResponse struct {
    ModelList []*Model `json:"model"`
    NextPageToken string `json:"nextPageToken"`
}


type TrainOperation struct {
    Id              string
    CreateTime      time.Time
    UpdateTime      time.Time
    ProgressPercent int
    ErrorCode       int
}

type TrainOperationReport struct {
    Running     []*TrainOperation
    Error       []*TrainOperation
    Cancelled   []*TrainOperation
}

type TranslationDatasetMetadata struct {
    SourceLanguageCode string `json:"sourceLanguageCode"`
    TargetLanguageCode string `json:"targetLanguageCode"`
}

type Dataset struct {
   Name                         string `json:"name"`
   DisplayName                  string `json:"displayName"`
   CreateTime                   time.Time `json:"createTime"`
   ExampleCount                 int `json:"exampleCount"`
   UpdateTime                   time.Time `json:"updateTime"`
   TranslationDatasetMetadata   TranslationDatasetMetadata `json:"translationDatasetMetadata"`
}

type ListDatasetAPIResponse struct {
    DatasetList     []*Dataset `json:"datasets"`
    NextPageToken   string `json:"nextPageToken"`
}

// Request functions
func GetClient() (*http.Client, error) {

    // Set client with oauth
    data, err := ioutil.ReadFile("./auth/auth.json")
    if err != nil {
        return nil, err
    }
    
    // Set proper spaces
    conf, err := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/cloud-platform")
    if err != nil {
        return nil, err
    }

    return conf.Client(oauth2.NoContext), nil
}

func ProjectNumberRequest(infoLog, errorLog *log.Logger, projectId string) (string, error) {
    defaultValue := ""

    url := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v1/projects/%s", projectId)

    client, err := GetClient()
    if err != nil {
        return defaultValue, err
    }

    req, err := http.NewRequest("GET", url, nil)

    response, err := client.Do(req)
    if err != nil {
        return defaultValue, err
    }
    defer response.Body.Close()

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return defaultValue, err
    }

    var o map[string]*json.RawMessage
    err = json.Unmarshal(body, &o)
    if(err != nil){
        return defaultValue, err
    }

    var str string
    err = json.Unmarshal(*o["projectNumber"], &str)
    if(err != nil){
        return defaultValue, err
    }

    return str, nil
}


func ListModelsRequest(infoLog, errorLog *log.Logger, projectId string) ([]*Model, error) {
    var defaultValue []*Model

    projectNumber, err := ProjectNumberRequest(infoLog, errorLog, projectId)
    if err != nil {
        return defaultValue, err
    }

    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/projects/%s/locations/us-central1/models", projectNumber)

    client, err := GetClient()
    if err != nil {
        return defaultValue, err
    }

    req, err := http.NewRequest("GET", url, nil)

    // Debug request
    dump, err := httputil.DumpRequestOut(req, true)
    if err != nil {
        return defaultValue, err
    }

    infoLog.Printf("%s", dump)

    response, err := client.Do(req)
    if err != nil {
        return defaultValue, err
    }
    defer response.Body.Close()

    // Debug response
    dump, err = httputil.DumpResponse(response, true)
    if err != nil {
        return defaultValue, err
    }
    infoLog.Printf("%s", dump)

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return defaultValue, err
    }

    t := new(ListModelAPIResponse)
    err = json.Unmarshal(body, &t)
    if(err != nil){
        return defaultValue, err
    }

    return t.ModelList, nil
}


func TranslateRequest(infoLog, errorLog *log.Logger, modelName, sourceText string) (string, error) {
    defaultValue := ""

    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/%s:predict", modelName)

    client, err := GetClient()
    if err != nil {
        return defaultValue, err
    }

    jsonStr := []byte(fmt.Sprintf(`{"payload": {"textSnippet": { "content": '%s'}}}`, sourceText))
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

    // Debug request
    dump, err := httputil.DumpRequestOut(req, true)
    if err != nil {
        return defaultValue, err
    }

    infoLog.Printf("%s", dump)
    
    response, err := client.Do(req)
    if err != nil {
        return defaultValue, err
    }
    defer response.Body.Close()

    // Debug response    
    dump, err = httputil.DumpResponse(response, true)
    if err != nil {
        return defaultValue, err
    }
    infoLog.Printf("%s", dump)

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return defaultValue, err
    }

    t := new(TranslationAPIResponse)
    err = json.Unmarshal(body, &t)
    if(err != nil){
        return defaultValue, err
    }

    return t.PayloadList[0].Translation.TranslatedContent.Content, nil
}

func CheckTranslationReply(infoLog, errorLog *log.Logger, response *http.Response) ([]byte, error) {
    defer response.Body.Close()

    statusCode := response.StatusCode
    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return nil, err
    }

    if statusCode == 200 {
        return body, nil
    }

    errorMessage := gjson.GetBytes(body, "error")
    if errorMessage.Get("code").Exists() && errorMessage.Get("message").Exists() {
        err = errors.New(fmt.Sprintf("automl: Error code [%s][%s]",
                                     errorMessage.Get("code").String(), errorMessage.Get("errorMessage").String()))
    } else{
        err = errors.New("automl: Undefined Google API Error")
    }

    return nil, err
}

func MakeTranslationRequest(infoLog, errorLog *log.Logger, urlQuery string, jsonStr []byte, totalTries int) ([]byte, error) {
    var err error;
    var body []byte;

    client, err := GetClient()
    if err != nil {
        return nil, err
    }

    for currentTry := 0; currentTry < totalTries; currentTry++ {
        req, err := http.NewRequest("POST", urlQuery, bytes.NewBuffer(jsonStr))

        response, err := client.Do(req)
        if err != nil {
            return nil, err
        }

        body, err = CheckTranslationReply(infoLog, errorLog, response)
        if err == nil { break }

        currentTry += 1
        time.Sleep(6 * time.Second)
    }
    return body, err
}

func StringToLines(s string) (lines []string, err error) {
    scanner := bufio.NewScanner(strings.NewReader(s))
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    err = scanner.Err()
    return
}

func TranslateBaseRequest(infoLog, errorLog *log.Logger, modelName, source, target, sourceText string) (string, error) {
    defaultValue := ""

    urlQuery := "https://translation.googleapis.com/language/translate/v2"

    totalText, err := url.QueryUnescape(sourceText)
    if err != nil {
        return defaultValue, err
    }

    lines, err := StringToLines(totalText)
    if err != nil {
        return defaultValue, err
    }

    var paragraphs = ""
    for _, partialText := range lines {
        if partialText == "" {
            paragraphs = fmt.Sprintf(`%s, "q": "%s"`, paragraphs, ".-1-.")
        }else {
            paragraphs = fmt.Sprintf(`%s, "q": "%s"`, paragraphs, partialText)
        }
    }

    jsonStr := []byte(fmt.Sprintf(`{"format": "text", "source": "%s", "target": "%s" %s}`, source, target, paragraphs))

    var totalTries = 10
    body, err := MakeTranslationRequest(infoLog, errorLog, urlQuery, jsonStr, totalTries)
    if err != nil {
        return defaultValue, err
    }

    var translatedText string = ""

    translations := gjson.GetBytes(body, "data.translations")
    translations.ForEach(func(key, translation gjson.Result) bool {
        // If it's a train model operation
        if translation.Get("translatedText").Exists(){

            //partialTranslatedText := strings.Trim(translation.Get("translatedText").String(), "\n")
            partialTranslatedText := translation.Get("translatedText").String()

            if partialTranslatedText == ".-1-." {
                translatedText += partialTranslatedText
            }else {
                translatedText += partialTranslatedText + "\n"
            }

        }
        return true // continue iterating
    })

    translatedText = strings.Replace(strings.TrimRight(translatedText, "\n"), ".-1-.", "\n", -1)

    //infoLog.Println(fmt.Sprintf("%s", strings.Replace(translatedText, "\n", "\\n", -1)))

    return translatedText, nil
}


func ListTrainOperationsRequest(infoLog, errorLog *log.Logger, projectId string) (*TrainOperationReport, error) {
    var defaultValue *TrainOperationReport

    projectNumber, err := ProjectNumberRequest(infoLog, errorLog, projectId)
    if err != nil {
        return defaultValue, err
    }

    name := fmt.Sprintf("projects/%s/locations/us-central1", projectNumber)
    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/%s/operations", name)

    client, err := GetClient()
    if err != nil {
        return defaultValue, err
    }

    req, err := http.NewRequest("GET", url, nil)

    response, err := client.Do(req)
    if err != nil {
        return defaultValue, err
    }
    defer response.Body.Close()

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return defaultValue, err
    }

    report := &TrainOperationReport{
        Running: []*TrainOperation{},
        Error: []*TrainOperation{},
        Cancelled: []*TrainOperation{},
    }

    operations := gjson.GetBytes(body, "operations")
    operations.ForEach(func(key, operation gjson.Result) bool {
        // If it's a train model operation
        if operation.Get("metadata.createModelDetails").Exists(){
            operationSplit := strings.Split(operation.Get("name").String(), "/")
            operationId := operationSplit[len(operationSplit)-1]

            createTime := operation.Get("metadata.createTime").Time()
            updateTime := operation.Get("metadata.updateTime").Time()

            progressPercent := 0
            errorCode := 0

            op := TrainOperation{
                Id:                 operationId,
                CreateTime:         createTime,
                UpdateTime:         updateTime,
                ProgressPercent:    progressPercent,
                ErrorCode:          errorCode,
            }

            if operation.Get("done").Exists() {
                e := operation.Get("error.code")
                if e.Exists(){
                    e.Int()
                    progressPercent = int(operation.Get("progressPercent").Int())

                    // User cancelled
                    if errorCode == 1{
                        report.Error = append(report.Error, &op)
                    // Stopped due to error
                    }else{
                        report.Cancelled = append(report.Cancelled, &op)
                    }
                }
            // Running operations
            }else{
                report.Running = append(report.Running, &op)
            }
        }

        return true // keep iterating
    })

    infoLog.Printf("%#v", report)

    return report, nil
}


func ListDatasetsRequest(infoLog, errorLog *log.Logger, projectId string) ([]*Dataset, error) {
    var defaultValue []*Dataset

    projectNumber, err := ProjectNumberRequest(infoLog, errorLog, projectId)
    if err != nil {
        return defaultValue, err
    }

    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/projects/%s/locations/us-central1/datasets", projectNumber)

    client, err := GetClient()
    if err != nil {
        return defaultValue, err
    }
    req, err := http.NewRequest("GET", url, nil)

    response, err := client.Do(req)
    if err != nil {
        return defaultValue, err
    }
    defer response.Body.Close()

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return defaultValue, err
    }

    t := new(ListDatasetAPIResponse)
    err = json.Unmarshal(body, &t)
    if(err != nil){
        return defaultValue, err
    }

    infoLog.Printf("%#v", t.DatasetList)

    return t.DatasetList, nil
}


func DeleteDatasetRequest(infoLog, errorLog *log.Logger, projectId, datasetId string) error {
    projectNumber, err := ProjectNumberRequest(infoLog, errorLog, projectId)
    if err != nil {
        return err
    }

    datasetName := fmt.Sprintf("projects/%s/locations/us-central1/datasets/%s", projectNumber, datasetId)
    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/%s", datasetName)

    client, err := GetClient()
    if err != nil {
        return err
    }
    req, err := http.NewRequest("DELETE", url, nil)

    response, err := client.Do(req)
    if err != nil {
        return err
    }
    defer response.Body.Close()

    return nil
}


func TrainModelRequest(infoLog, errorLog *log.Logger, projectId, datasetId, displayName string) error {
    projectNumber, err := ProjectNumberRequest(infoLog, errorLog, projectId)
    if err != nil {
        return err
    }

    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/projects/%s/locations/us-central1/models", projectNumber)

    client, err := GetClient()
    if err != nil {
        return err
    }

    jsonStr := []byte(fmt.Sprintf(`
        {
        "displayName": "%s",
        "datasetId": "%s",
        "translationModelMetadata": {
            "baseModel" : ""
            },
        }`, displayName, datasetId))

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

    response, err := client.Do(req)
    if err != nil {
        return err
    }
    defer response.Body.Close()

    // Debug response
    dump, err := httputil.DumpResponse(response, true)
    if err != nil {
        return err
    }
    infoLog.Printf("%s", dump)

    return nil
}


func GetDatasetsDisplayName(datasets []*Dataset) []*string {
    names := []*string{}

    for _, dataset := range datasets {
        names = append(names, &dataset.DisplayName)
    }

    return names
}


func GetModelsDisplayName(models []*Model) []*string {
    names := []*string{}

    for _, model := range models {
        names = append(names, &model.DisplayName)
    }

    return names
}


func DeleteModelRequest(infoLog, errorLog *log.Logger, projectId, modelId string) error {
    projectNumber, err := ProjectNumberRequest(infoLog, errorLog, projectId)
    if err != nil {
        return err
    }

    datasetName := fmt.Sprintf("projects/%s/locations/us-central1/models/%s", projectNumber, modelId)
    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/%s", datasetName)

    client, err := GetClient()
    if err != nil {
        return err
    }
    req, err := http.NewRequest("DELETE", url, nil)

    response, err := client.Do(req)
    if err != nil {
        return err
    }
    defer response.Body.Close()

    // Debug response
    dump, err := httputil.DumpResponse(response, true)
    if err != nil {
        return err
    }
    infoLog.Printf("%s", dump)

    return nil
}


func CancelTrainRequest(infoLog, errorLog *log.Logger, projectId, modelId string) error {
    projectNumber, err := ProjectNumberRequest(infoLog, errorLog, projectId)
    if err != nil {
        return err
    }

    operationName := fmt.Sprintf("projects/%s/locations/us-central1/operations/%s", projectNumber, modelId)
    url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/%s:cancel", operationName)

    client, err := GetClient()
    if err != nil {
        return err
    }
    req, err := http.NewRequest("POST", url, nil)

    response, err := client.Do(req)
    if err != nil {
        return err
    }
    defer response.Body.Close()

    // Debug response
    dump, err := httputil.DumpResponse(response, true)
    if err != nil {
        return err
    }
    infoLog.Printf("%s", dump)

    return nil
}