package eventstore

import "time"

type Event interface {
	AggregateID() string
	AggregateType() string
	EventType() string
	Version() int
	Timestamp() time.Time
	Metadata() map[string]string
}

//nolint:recvcheck // TODO: migrate all receivers to pointer receivers to be consistent with SetEnvelope
type BaseEvent struct {
	aggID   string
	aggType string
	evType  string
	version int
	ts      time.Time
	meta    map[string]string
}

func NewBaseEvent(aggID, aggType, evType string, version int, meta map[string]string) BaseEvent {
	return BaseEvent{aggID: aggID, aggType: aggType, evType: evType,
		version: version, ts: time.Now().UTC(), meta: meta}
}

func (b BaseEvent) AggregateID() string         { return b.aggID }
func (b BaseEvent) AggregateType() string       { return b.aggType }
func (b BaseEvent) EventType() string           { return b.evType }
func (b BaseEvent) Version() int                { return b.version }
func (b BaseEvent) Timestamp() time.Time        { return b.ts }
func (b BaseEvent) Metadata() map[string]string { return b.meta }

// SetEnvelope populates the unexported envelope fields after deserialization.
func (b *BaseEvent) SetEnvelope(aggID, aggType, evType string, version int, ts time.Time, meta map[string]string) {
	b.aggID = aggID
	b.aggType = aggType
	b.evType = evType
	b.version = version
	b.ts = ts
	b.meta = meta
}

// Enveloper is implemented by events embedding BaseEvent.
type Enveloper interface {
	SetEnvelope(aggID, aggType, evType string, version int, ts time.Time, meta map[string]string)
}
