package cache

import "github.com/tobyjsullivan/ues-sdk/event"

type EventCache struct {
    data map[event.EventID]*event.Event
}

func New() *EventCache {
    return &EventCache{
        data: make(map[event.EventID]*event.Event),
    }
}

func (c *EventCache) Get(id event.EventID) (*event.Event, bool) {
    e, ok := c.data[id]
    return e, ok
}

func (c *EventCache) Add(e *event.Event) {
    id := e.ID()
    if _, ok := c.data[id]; ok {
        return
    }
    c.data[id] = e
}
