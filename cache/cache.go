package cache

import "github.com/tobyjsullivan/ues-sdk/event"

type EventCache struct {
    data map[event.EventID]*event.Event
    list *list
    maxSize int
}

func New(maxSize int) *EventCache {
    return &EventCache{
        data: make(map[event.EventID]*event.Event),
        list: &list{},
        maxSize: maxSize,
    }
}

func (c *EventCache) Get(id event.EventID) (*event.Event, bool) {
    e, ok := c.data[id]
    if ok {
        c.list.remove(id)
        c.list.prepend(id)
    }
    return e, ok
}

func (c *EventCache) Add(e *event.Event) {
    id := e.ID()
    if _, ok := c.data[id]; ok {
        return
    }
    c.data[id] = e
    c.list.prepend(id)

    if c.list.len() > c.maxSize {
        c.removeOldest()
    }
}

func (c *EventCache) removeOldest() {
    oldest, any := c.list.last()
    if !any {
        return
    }

    c.list.remove(oldest)
    delete(c.data, oldest)
}
