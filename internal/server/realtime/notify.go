// Package realtime broadcasts pre-computed dashboard payloads to SSE
// subscribers via PocketBase's SubscriptionsBroker.
//
// Topics are fan-out only — clients can only listen; the server sends
// the same payloads the REST endpoints would return, but pushed when
// data actually changes (after batch commits, after snapshot jobs).
package realtime

import (
	"encoding/json"
	"sync"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

// Broadcast serializes payload and sends it to every client subscribed
// to the given topic. Non-blocking per client: subscriptions.Client.Send
// already recovers from a closed channel and does the channel write.
//
// Returns early without computing JSON if there are no clients at all.
func Broadcast(app core.App, topic string, payload any) error {
	broker := app.SubscriptionsBroker()
	if broker.TotalClients() == 0 {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := subscriptions.Message{Name: topic, Data: data}

	chunks := broker.ChunkedClients(300)
	var wg sync.WaitGroup
	for _, chunk := range chunks {
		wg.Add(1)
		go func(clients []subscriptions.Client) {
			defer wg.Done()
			for _, client := range clients {
				if client.HasSubscription(topic) {
					client.Send(msg)
				}
			}
		}(chunk)
	}
	wg.Wait()
	return nil
}

// HasSubscribers reports whether at least one connected client is
// subscribed to topic. Used to skip expensive payload computation when
// nobody is listening.
func HasSubscribers(app core.App, topic string) bool {
	broker := app.SubscriptionsBroker()
	if broker.TotalClients() == 0 {
		return false
	}
	for _, c := range broker.Clients() {
		if c.HasSubscription(topic) {
			return true
		}
	}
	return false
}
