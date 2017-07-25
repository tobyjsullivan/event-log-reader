package event

import "encoding/base64"

type EventData []byte

func (d *EventData) String() string {
    return base64.StdEncoding.EncodeToString([]byte(*d))
}

func ParseData(s string) (EventData, error) {
    b, err := base64.StdEncoding.DecodeString(s)
    if err != nil {
        return EventData{}, err
    }

    return EventData(b), nil
}
