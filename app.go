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
    eventLog "github.com/tobyjsullivan/event-log-reader/log"
    "encoding/json"
    "encoding/base64"
    "github.com/tobyjsullivan/ues-sdk/event/reader"
    "github.com/tobyjsullivan/ues-sdk/event"
    "github.com/tobyjsullivan/event-log-reader/cache"
    "github.com/go-redis/redis"
)

const (
    CACHE_MAX_KEYS = 50000
)

var (
    logger     *log.Logger
    db         *sql.DB
    eventReader *reader.EventReader
    eventCache *cache.EventCache
    redisClient *redis.Client
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

    eventReader, err = reader.New(&reader.EventReaderConfig{
        ServiceUrl: os.Getenv("EVENT_READER_API"),
    })
    if err != nil {
        logger.Println("Error initializing Event Reader API.", err.Error())
        panic(err.Error())
    }

    eventCache = cache.New(CACHE_MAX_KEYS)

    redisClient = redis.NewClient(&redis.Options{
        Addr: fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOSTNAME"), os.Getenv("REDIS_PORT")),
        Password: os.Getenv("REDIS_PASSWORD"),
        DB: 0,
    })

    pong, err := redisClient.Ping().Result()
    logger.Println("Pong result:", pong, err)
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

    after := event.EventID{}
    afterParam := r.URL.Query().Get("after")
    if afterParam != "" {
        err := after.Parse(afterParam)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
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

    events, err := getEventHistory(headEventId, after)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    l := len(events)
    jsonEvents := make([]*eventJson, l)
    for i := 0; i < l; i++ {
        e := events[l - 1 - i]
        id := e.ID()

        jsonEvents[i] = &eventJson{
            EventID: id.String(),
            Type: e.Type,
            Data: base64.StdEncoding.EncodeToString(e.Data),
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
    Type string `json:"type"`
    Data string `json:"data"`
}

func getLogHead(conn *sql.DB, id eventLog.LogID) (event.EventID, error) {
    var head []byte
    err := conn.QueryRow(`SELECT head FROM logs WHERE ext_lookup_key=$1`, id[:]).Scan(&head)
    if err == sql.ErrNoRows {
        // Return the Zero Event if there is no record of the log (we treat an unknown log as an empty log)
        return event.EventID{}, nil
    }

    if err != nil {
        logger.Println("Error executing SELECT for log head lookup.", err.Error())
        return event.EventID{}, err
    }

    var out event.EventID
    copy(out[:], head)
    return out, nil
}

func getEventHistory(head event.EventID, last event.EventID) ([]*event.Event, error) {
    zero := event.EventID{}

    out := make([]*event.Event, 0)
    for head != last && head != zero {
        e, err := getEvent(head)
        if err != nil {
            return []*event.Event{}, err
        }

        out = append(out, e)
        head = e.PreviousEvent
    }

    return out, nil
}

func getEvent(id event.EventID) (*event.Event, error) {
    if e, ok := eventCache.Get(id); ok {
        return e, nil
    }

    if e, ok := redisGet(id); ok {
        return e, nil
    }

    e, err := eventReader.GetEvent(id)
    if err != nil {
        return nil, err
    }

    go addToCaches(e)

    return e, nil
}

func addToCaches(e *event.Event) {
    eventCache.Add(e)

    redisSet(e)
}

func redisSet(e *event.Event) {
    id := e.ID()

    redisClient.Set(id.String(), e.String(), 0)
}

func redisGet(id event.EventID) (*event.Event, bool) {
    res, err := redisClient.Get(id.String()).Result()
    if err != nil {
        if err != redis.Nil {
            logger.Println("Redis error:", err.Error())
        }
        return nil, false
    }

    var e event.Event
    err = e.Parse(res)
    if err != nil {
        logger.Println("Error deserializing redis result.", err.Error())
        return nil, false
    }
    return &e, true
}

type redisEventSerializer struct {
    PrevID string `json:"previousId"`
    Type string `json:"type"`
    Data string `json:"data"`
}
