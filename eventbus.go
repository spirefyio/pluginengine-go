package pluginengine

import "sync"

type Event struct {
    Name    string
    Payload []byte
}

type Listener func(event Event, callback func(response []byte, err error))

type EventBus struct {
    listeners map[string][]Listener
    mu        sync.RWMutex
}

func NewEventBus() *EventBus {
    return &EventBus{
        listeners: make(map[string][]Listener),
    }
}

func (bus *EventBus) RegisterListener(eventName string, listener Listener) {
    bus.mu.Lock()
    defer bus.mu.Unlock()
    bus.listeners[eventName] = append(bus.listeners[eventName], listener)
}


func (bus *EventBus) DispatchEvent(event Event, callback func(response []byte, err error)) {
    bus.mu.RLock()
    defer bus.mu.RUnlock()

    listeners, exists := bus.listeners[event.Name]
    if !exists {
        callback(nil, nil)
        return
    }

    var wg sync.WaitGroup
    for _, listener := range listeners {
        wg.Add(1)
        go func(listener Listener) {
            defer wg.Done()
            listener(event, callback)
        }(listener)
    }
}