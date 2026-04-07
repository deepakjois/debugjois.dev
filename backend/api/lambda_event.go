package main

import "encoding/json"

type eventType int

const (
	eventTypeAPIGateway eventType = iota
	eventTypeScheduled
	eventTypeDirect
)

// eventProbe unmarshals only the fields needed to classify a Lambda event.
type eventProbe struct {
	// API Gateway V2: uniquely identified by requestContext.http.
	RequestContext *struct {
		HTTP *struct{} `json:"http"`
	} `json:"requestContext"`

	// EventBridge / CloudWatch scheduled events: identified by source + detail-type.
	Source     string `json:"source"`
	DetailType string `json:"detail-type"`
}

// classifyEvent inspects a raw Lambda payload and determines its event source.
func classifyEvent(payload json.RawMessage) eventType {
	var probe eventProbe
	if err := json.Unmarshal(payload, &probe); err != nil {
		return eventTypeDirect
	}

	if probe.RequestContext != nil && probe.RequestContext.HTTP != nil {
		return eventTypeAPIGateway
	}

	if probe.Source != "" && probe.DetailType != "" {
		return eventTypeScheduled
	}

	return eventTypeDirect
}
