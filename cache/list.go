package cache

import "github.com/tobyjsullivan/ues-sdk/event"

type list struct {
    head *node
}

func (l *list) prepend(k event.EventID) {
    h := l.head
    l.head = &node{
        key: k,
        next: h,
    }
}

func (l *list) last() (event.EventID, bool) {
    n := l.head
    if n == nil {
        return event.EventID{}, false
    }
    for n.next != nil {
        n = n.next
    }
    return n.key, true
}

func (l *list) len() int {
    i := 0
    n := l.head
    for n != nil {
        i++
        n = n.next
    }
    return i
}

func (l *list) remove(k event.EventID) {
    prev := l.head
    n := prev.next
    for n != nil {
        if n.key == k {
            // Remove
            prev.next = n.next
            n = n.next
            continue
        }

        prev = n
        n = n.next
    }
}

type node struct {
    key event.EventID
    next *node
}

