package main

import (
	"container/list"
	"context"
	"sync"

	"nhooyr.io/websocket"
)

var (
	subscribers   = make(map[*websocket.Conn]interface{})
	subscribersMu = sync.RWMutex{}
)

func addSubscriber(subscriber *websocket.Conn) {
	subscribersMu.Lock()
	defer subscribersMu.Unlock()
	subscribers[subscriber] = nil
}

func delSubscriber(subscriber *websocket.Conn) {
	subscribersMu.Lock()
	defer subscribersMu.Unlock()
	delete(subscribers, subscriber)
}

func publish(ctx context.Context, msg []byte, record bool) (msgObj *list.Element) {
	subscribersMu.RLock()
	defer subscribersMu.RUnlock()

	if record {
		msgObj = history.PushBack(msg)
	}
	for history.Len() > *maxChatHistory {
		history.Remove(history.Front())
	}

	for s := range subscribers {
		s.Write(ctx, websocket.MessageBinary, msg)
	}

	return
}
