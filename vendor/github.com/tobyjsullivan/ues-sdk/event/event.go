package event

import "crypto/sha256"

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
