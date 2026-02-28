package order

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"

type OrderState struct {
	ID     string
	UserID string
	Status string
	Total  float64
	Active bool
}

type OrderCreated struct {
	eventstore.BaseEvent
	UserID string  `json:"user_id"`
	Total  float64 `json:"total"`
}

type OrderUpdated struct {
	eventstore.BaseEvent
	Status string  `json:"status"`
	Total  float64 `json:"total"`
}

type OrderDeleted struct {
	eventstore.BaseEvent
}

func applyOrder(s *OrderState, e eventstore.Event) {
	switch ev := e.(type) {
	case *OrderCreated:
		s.ID = ev.AggregateID()
		s.UserID = ev.UserID
		s.Total = ev.Total
		s.Status = "pending"
		s.Active = true
	case *OrderUpdated:
		s.Status = ev.Status
		s.Total = ev.Total
	case *OrderDeleted:
		s.Active = false
	}
}

func NewOrderAggregate(id string) *eventstore.Aggregate[OrderState] {
	return eventstore.New[OrderState](id, applyOrder)
}
