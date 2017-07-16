package reader

import (
    "github.com/tobyjsullivan/event-store.v3/events"
    "net/url"
    "net/http"
    "encoding/json"
    "encoding/base64"
)

type EventReader struct {
    readerApi *url.URL
}

func New(apiUrl string) (*EventReader, error) {
    api, err := url.Parse(apiUrl)
    if err != nil {
        return nil, err
    }

    return &EventReader{
        readerApi: api,
    }, nil
}

type eventReaderResp struct {
    Previous string `json:"previous"`
    Type string `json:"type"`
    Data string `json:"data"`
}

func (r *EventReader) ReadEvent(eventId events.EventID) (*events.Event, error) {
    strEventId := eventId.String()

    endpointUrl := r.readerApi.ResolveReference(&url.URL{Path: "./events/"+strEventId})
    resp, err := http.Get(endpointUrl.String())
    if err != nil {
        return nil, err
    }

    decoder := json.NewDecoder(resp.Body)
    defer resp.Body.Close()

    var parsedResp eventReaderResp
    err = decoder.Decode(&parsedResp)
    if err != nil {
        return nil, err
    }

    prevId := events.EventID{}
    err = prevId.Parse(parsedResp.Previous)
    if err != nil {
        return nil, err
    }

    parsedData, err := base64.StdEncoding.DecodeString(parsedResp.Data)
    if err != nil {
        return nil, err
    }

    return &events.Event{
        PreviousEvent: prevId,
        Type: parsedResp.Type,
        Data: parsedData,
    }, nil
}
