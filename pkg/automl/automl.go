package automl

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/http/httputil"
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
    ModelList []Model `json:"model"`
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

    models := []*Model{}
    for _, element := range t.ModelList {
        models = append(models, &element)
    }

    return models, nil
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

            createTime := operation.Get("createTime").Time()
            updateTime := operation.Get("updateTime").Time()

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