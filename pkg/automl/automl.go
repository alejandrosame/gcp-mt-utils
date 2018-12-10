package automl

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/http/httputil"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

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


func TranslateRequest(infoLog, errorLog *log.Logger, modelName, sourceText string) (string, error) {
	
	url := fmt.Sprintf("https://automl.googleapis.com/v1beta1/%s:predict", modelName) 

    client, err := GetClient()
    if err != nil {
        return "", err
    }

    //client := &http.Client{}
    jsonStr := []byte(fmt.Sprintf(`{"payload": {"textSnippet": { "content": '%s'}}}`, sourceText))
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

    // Debug request
    dump, err := httputil.DumpRequestOut(req, true)
    if err != nil {
        return "", err
    }

    infoLog.Printf("%s", dump)
    
    response, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer response.Body.Close()

    // Debug response    
    dump, err = httputil.DumpResponse(response, true)
    if err != nil {
        return "", err
    }
    infoLog.Printf("%s", dump)

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return "", err
    }

    t := new(TranslationAPIResponse)
    err = json.Unmarshal(body, &t)
    if(err != nil){
    	return "", err
    }

    return t.PayloadList[0].Translation.TranslatedContent.Content, nil
}