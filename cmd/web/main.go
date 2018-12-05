package main

import (
    "database/sql"
    "flag"
    "html/template"
    "log"
    "net/http"
    "os"

    "github.com/alejandrosame/gcp-mt-utils/pkg/models/mysql"

    _ "github.com/go-sql-driver/mysql"
)

type application struct {
    errorLog *log.Logger
    infoLog  *log.Logger
    pairs *mysql.PairModel
    templateCache map[string]*template.Template
}

func main() {

    addr := flag.String("addr", ":4000", "HTTP network address")
    dsn := flag.String("dsn", "web:123456@/gcp_mt_pairs?parseTime=true", "MySQL data source name")
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

    // Add objects to app struct
    app := &application{
        errorLog: errorLog,
        infoLog:  infoLog,
        pairs: &mysql.PairModel{DB: db},
        templateCache: templateCache,
    }

    srv := &http.Server{
        Addr:     *addr,
        ErrorLog: errorLog,
        Handler:  app.routes(),
    }

    infoLog.Printf("Starting server on %s", *addr)
    err = srv.ListenAndServe()
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