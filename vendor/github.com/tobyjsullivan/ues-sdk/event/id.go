package event

import (
    "encoding/hex"
)

type EventID [32]byte

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
