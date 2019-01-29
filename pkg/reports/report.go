package reports

import (
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "strings"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models"

    "github.com/ipinfo/go-ipinfo/ipinfo"
)


type GeoIP struct {
    IP          string
    City        string
    Region      string
    Country     string
    Location    string
    Phone       string
    Postal      string
}


func ExtractIP(infoLog, errorLog *log.Logger, r *http.Request) (string, error){
    infoLog.Println(fmt.Sprintf("IP-FORWARDED: %s", r.Header.Get("X-Forwarded-For")))
    ipForwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ", ")[0]
    if ipForwarded != ""{
        return ipForwarded, nil
    }
    return strings.Split(r.RemoteAddr, ":")[0], nil
}

func CheckGeoElement(value string) (string){

    if value == "" {
        return "Cannot locate IP"
    }
    return value
}

func GetGeoInfo(infoLog, errorLog *log.Logger, ipString string) (*GeoIP, error){
    token := os.Getenv("IPINFO_API_KEY")
    tp := ipinfo.AuthTransport{
        Token: token,
    }

    apiInfoClient := ipinfo.NewClient(tp.Client())

    ip := net.ParseIP(ipString)
    info, err := apiInfoClient.GetGeo(ip)
    if err != nil {
        return nil, err
    }

    geoIP := GeoIP{IP: ipString,
                   City: CheckGeoElement(info.City),
                   Region: CheckGeoElement(info.Region),
                   Country: CheckGeoElement(info.Country),
                   Location: CheckGeoElement(info.Location),
                   Phone: CheckGeoElement(info.Phone),
                   Postal: CheckGeoElement(info.Postal)}

    return &geoIP, nil
}

func GenerateReportFromRequest(infoLog, errorLog *log.Logger, r *http.Request, user *models.User,
                               characterCount int, title, requestDate string) (string, string, error){

    ip, err := ExtractIP(infoLog, errorLog, r)
    if err != nil {
        return "", "", err
    }

    geoIP, err := GetGeoInfo(infoLog, errorLog, ip)
    if err != nil {
        return "", "", err
    }

    report := map[string]string{}

    report["User Name"] = user.Name
    report["User Email"] = user.Email

    report["Device"] = r.Header.Get("User-Agent")

    report["IP"] = geoIP.IP
    report["City"] = geoIP.City
    report["Region"] = geoIP.Region
    report["Country"] = geoIP.Country
    report["Coordinates"] = geoIP.Location
    report["Zipcode"] = geoIP.Postal

    report["Title"] = title
    report["Translation Request Date"] = requestDate
    report["Character Count"] = fmt.Sprintf("%d", characterCount)

    keys := []string{"User Name", "User Email", "Device", "IP", "City", "Region", "Country", "Coordinates", "Zipcode",
                     "Title", "Translation Request Date", "Character Count"}

    plainContent := ""
    htmlContent := ""

    for _, k := range keys {
        plainContent += fmt.Sprintf("-%s: %s\n", k, report[k])
        htmlContent += fmt.Sprintf("<strong>-%s: </strong>%s<br>", k, report[k])
    }

    return strings.TrimRight(plainContent, "\n"), strings.TrimRight(htmlContent, "\n"), nil
}