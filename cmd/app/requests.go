package app

import (
	"context"
	"delivery/types"
	"fmt"
	"net/http"
	"time"
)

// composes http.Request from Message, executes it with App's http.Client.Do,
// and returns a corresponding LogEntry for the event.
// Request timeout determined by httpTimeout config variable used in Client instantiation.
func (a *app) makeRequest(ctx context.Context, m *types.Message) (*types.LogEntry, error) {
	req, err := http.NewRequestWithContext(ctx, m.Method, m.URL, nil)
	if err != nil {
		a.Log.Error(fmt.Sprintf("error from http.NewReqWCtx: %v", err))
		// handle error
	}
	
	deliveryTime := time.Now()
	res, err := a.client.Do(req)
	responseTime := time.Now()

	if err != nil {
		a.Log.Error(fmt.Sprintf("req failed waiting %v, timeout %v: %v", time.Since(deliveryTime), a.client.Timeout, err))
		return nil, err
	}

	body := []byte{}
	_, err = res.Body.Read(body)
	if err != nil {
		a.Log.Error(fmt.Sprintf("failed to parse response body %v", err))
		// LogEntry otherwise created with empty body
	}

	return types.NewLogEntry(deliveryTime, responseTime, res.StatusCode, string(body)), nil
}
