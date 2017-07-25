package event

import (
    "crypto/sha256"
    "encoding/json"
)

type Event struct {
    PreviousEvent EventID
    Type string
    Data EventData
}

func (e *Event) ID() EventID {
    canonical := []byte{}
    canonical = append(canonical, e.PreviousEvent[:]...)
    canonical = append(canonical, []byte(e.Type)...)
    canonical = append(canonical, e.Data[:]...)

    out := sha256.Sum256(canonical)
    id := EventID(out)

    return id
}

func (e *Event) String() string {
    prevId := e.PreviousEvent
    data := e.Data

    ser := eventFormat{
        PreviousID: prevId.String(),
        Type: e.Type,
        Data: data.String(),
    }
    b, err := json.Marshal(&ser)
    if err != nil {
        panic(err.Error())
    }

    return string(b)
}

func (e *Event) Parse(s string) error {
    var ser eventFormat
    err := json.Unmarshal([]byte(s), &ser)
    if err != nil {
        return err
    }

    data, err := ParseData(ser.Data)
    if err != nil {
        return err
    }

    e.PreviousEvent.Parse(ser.PreviousID)
    e.Type = ser.Type
    e.Data = data

    return nil
}

type eventFormat struct {
    PreviousID string `json:"previousId"`
    Type string `json:"type"`
    Data string `json:"data"`
}
