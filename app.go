package main

import (
    "os"

    "fmt"
    "net/http"

    "github.com/urfave/negroni"
    "github.com/gorilla/mux"
    "log"
    "database/sql"

    _ "github.com/lib/pq"
    eventLog "github.com/tobyjsullivan/event-log/log"
    "github.com/tobyjsullivan/event-store.v3/events"
    "encoding/json"
)


var (
    logger     *log.Logger
    db         *sql.DB
)

func init() {
    logger = log.New(os.Stdout, "[svc] ", 0)

    pgHostname := os.Getenv("PG_HOSTNAME")
    pgUsername := os.Getenv("PG_USERNAME")
    pgPassword := os.Getenv("PG_PASSWORD")
    pgDatabase := os.Getenv("PG_DATABASE")

    dbConnOpts := fmt.Sprintf("host='%s' user='%s' dbname='%s' password='%s' sslmode=disable",
        pgHostname, pgUsername, pgDatabase, pgPassword)

    logger.Println("Connecting to DB...")
    var err error
    db, err = sql.Open("postgres", dbConnOpts)
    if err != nil {
        logger.Println("Error initializing connection to Postgres DB.", err.Error())
        panic(err.Error())
    }
}


func main() {
    r := buildRoutes()

    n := negroni.New()
    n.UseHandler(r)

    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }

    n.Run(":" + port)
}

func buildRoutes() http.Handler {
    r := mux.NewRouter()
    r.HandleFunc("/", statusHandler).Methods("GET")
    r.HandleFunc("/logs/{logId}", readLogHandler).Methods("GET")

    return r
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "The service is online!\n")
}

func readLogHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    logIdParam := vars["logId"]
    if logIdParam == "" {
        http.Error(w, "Must supply logId in path.", http.StatusBadRequest)
        return
    }

    logId := eventLog.LogID{}
    err := logId.Parse(logIdParam)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    headEventId, err := getLogHead(db, logId)
    if err == sql.ErrNoRows {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    } else if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    resp := &jsonResponse{
        Data: &readLogResponse{
            LogID: logId.String(),
            Head: headEventId.String(),
        },
    }
    encoder := json.NewEncoder(w)
    err = encoder.Encode(resp)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

type jsonResponse struct {
    Data interface{} `json:"data,omitempty"`
    Error string `json:"error,omitempty"`
}

type readLogResponse struct {
    LogID string `json:"logId"`
    Head string `json:"head"`
}

func getLogHead(conn *sql.DB, id eventLog.LogID) (events.EventID, error) {
    var head []byte
    err := conn.QueryRow(`SELECT head FROM logs WHERE ext_lookup_key=$1`, id[:]).Scan(&head)
    if err != nil {
        logger.Println("Error executing SELECT for log head lookup.", err.Error())
        return events.EventID{}, err
    }

    var out events.EventID
    copy(out[:], head)
    return out, nil
}
