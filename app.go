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
    "encoding/json"
    "encoding/base64"
    "github.com/tobyjsullivan/ues-sdk/event/reader"
    "github.com/tobyjsullivan/ues-sdk/event"
)


var (
    logger     *log.Logger
    db         *sql.DB
    eventReader *reader.EventReader
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

    eventReader, err = reader.New(os.Getenv("EVENT_READER_API"))
    if err != nil {
        logger.Println("Error initializing Event Reader API.", err.Error())
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
    r.HandleFunc("/logs/{logId}/events", readEventsHandler).Methods("GET")

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

func readEventsHandler(w http.ResponseWriter, r *http.Request) {
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

    events, err := getEventHistory(headEventId)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    jsonEvents := make([]*eventJson, len(events))
    for i, e := range events {
        id := e.ID()
        strId := id.String()
        strPrevId := e.PreviousEvent.String()
        strData := base64.StdEncoding.EncodeToString(e.Data)

        jsonEvents[i] = &eventJson{
            EventID: strId,
            PrevID: strPrevId,
            Type: e.Type,
            Data: strData,
        }
    }

    resp := &jsonResponse{
        Data: &readEventsResponse{
            Events: jsonEvents,
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

type readEventsResponse struct {
    Events []*eventJson `json:"events"`
}

type eventJson struct {
    EventID string `json:"eventId"`
    PrevID string `json:"previousEventId"`
    Type string `json:"type"`
    Data string `json:"data"`
}

func getLogHead(conn *sql.DB, id eventLog.LogID) (event.EventID, error) {
    var head []byte
    err := conn.QueryRow(`SELECT head FROM logs WHERE ext_lookup_key=$1`, id[:]).Scan(&head)
    if err != nil {
        logger.Println("Error executing SELECT for log head lookup.", err.Error())
        return event.EventID{}, err
    }

    var out event.EventID
    copy(out[:], head)
    return out, nil
}

func getEventHistory(eventId event.EventID) ([]*event.Event, error) {
    zero := event.EventID{}

    out := make([]*event.Event, 0)
    for eventId != zero {
        e, err := eventReader.GetEvent(eventId)
        if err != nil {
            return []*event.Event{}, err
        }

        out = append(out, e)
        eventId = e.PreviousEvent
    }

    return out, nil
}
