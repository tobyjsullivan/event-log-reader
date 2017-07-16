package events

import (
    "crypto/sha256"
    "encoding/hex"
)

type EventID [32]byte

func NewEventID() EventID {
    var id [32]byte
    return EventID(id)
}

func (id *EventID) Parse(s string) error {
    in, err := hex.DecodeString(s)
    if err != nil {
        return err
    }

    copy(id[:], in)
    return nil
}

func (id *EventID) String() string {
    b := [32]byte(*id)
    return hex.EncodeToString(b[:])
}

type Event struct {
    PreviousEvent EventID
    Type string
    Data []byte
}

func (e *Event) ID() EventID {
    canonical := []byte{}
    canonical = append(canonical, e.PreviousEvent[:]...)
    canonical = append(canonical, []byte(e.Type)...)
    canonical = append(canonical, e.Data...)

    out := sha256.Sum256(canonical)
    id := EventID(out)

    return id
}
