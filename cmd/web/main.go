package main

import (
    "crypto/tls"
    "database/sql"
    "flag"
    "html/template"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models/mysql"

    _ "github.com/go-sql-driver/mysql"
    "github.com/golangcollege/sessions"
)

type contextKey string

var contextKeyUser = contextKey("user")
var contextKeyLanguages = contextKey("languages")

type application struct {
    errorLog      *log.Logger
    infoLog       *log.Logger
    maxUploadSize *int64
    pairs         *mysql.PairModel
    session       *sessions.Session
    templateCache map[string]*template.Template
    uploadPath    *string
    users         *mysql.UserModel
    invitations   *mysql.InvitationModel
    reports       *mysql.ReportModel
}

func main() {

    // Non string default values
    var defaultMaxUploadSize int64 = 2 * 1024 // 2 MB

    addr := flag.String("addr", ":4000", "HTTP network address")
    dsn := flag.String("dsn", "web:123456@/gcp_mt_pairs?parseTime=true", "MySQL data source name")
    secret := flag.String("secret", "s6Ndh+pPbnzHbS*+9Pk8qGWhTzbpa@ge", "Secret key")
    maxUploadSize := flag.Int64("max-upload-size", defaultMaxUploadSize, "File max upload size (MB)")
    uploadPath := flag.String("upload-path", "./tmp", "Upload file tmp folder")
    flag.Parse()

    infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
    errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

    // Init objects
    db, err := openDB(*dsn)
    if err != nil {
        errorLog.Fatal(err)
    }
    defer db.Close()

    templateCache, err := newTemplateCache()
    if err != nil {
        errorLog.Fatal(err)
    }

    // Sessions always expires after 10 minutes
    session := sessions.New([]byte(*secret))
    session.Lifetime = 60 * time.Minute
    session.Secure = true
    session.SameSite = http.SameSiteStrictMode

    // Add objects to app struct
    app := &application{
        errorLog:      errorLog,
        infoLog:       infoLog,
        maxUploadSize: maxUploadSize,
        pairs:         &mysql.PairModel{DB: db},
        session:       session,
        templateCache: templateCache,
        uploadPath:    uploadPath,
        users:         &mysql.UserModel{DB: db},
        invitations:   &mysql.InvitationModel{DB: db},
        reports:       &mysql.ReportModel{DB: db},
    }

    tlsConfig := &tls.Config{
        PreferServerCipherSuites: true,
        CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
    }

    srv := &http.Server{
        Addr:         *addr,
        ErrorLog:     errorLog,
        Handler:      app.routes(),
        TLSConfig:    tlsConfig,
        IdleTimeout:  time.Minute,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 5 * time.Minute,
    }

    infoLog.Printf("Starting server on %s", *addr)
    err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
    errorLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    if err = db.Ping(); err != nil {
        return nil, err
    }
    return db, nil
}