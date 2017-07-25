package reader

import (
    "net/url"
    "github.com/tobyjsullivan/ues-sdk/event"
    "net/http"
    "encoding/json"
)

type EventReader struct {
    apiBase *url.URL
}

type EventReaderConfig struct {
    ServiceUrl string
}

func New(conf *EventReaderConfig) (*EventReader, error) {
    api, err := url.Parse(conf.ServiceUrl)
    if err != nil {
        return nil, err
    }

    return &EventReader{
        apiBase: api,
    }, nil
}

func (svc *EventReader) GetEvent(id event.EventID) (*event.Event, error) {
    strEventId := id.String()

    endpointUrl := svc.apiBase.ResolveReference(&url.URL{Path: "./events/"+strEventId})
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

    prevId := event.EventID{}
    err = prevId.Parse(parsedResp.Previous)
    if err != nil {
        return nil, err
    }

    parsedData, err := event.ParseData(parsedResp.Data)
    if err != nil {
        return nil, err
    }

    return &event.Event{
        PreviousEvent: prevId,
        Type: parsedResp.Type,
        Data: parsedData,
    }, nil
}

type eventReaderResp struct {
    Previous string `json:"previous"`
    Type string `json:"type"`
    Data string `json:"data"`
}
