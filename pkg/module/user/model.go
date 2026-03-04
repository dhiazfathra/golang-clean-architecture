package user

import "github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"

type UserState struct {
	ID       string
	Email    string
	PassHash string
	Active   bool
	RoleIDs  []string
}

type UserCreated struct {
	eventstore.BaseEvent
	Email    string `json:"email"`
	PassHash string `json:"pass_hash"`
}

type EmailChanged struct {
	eventstore.BaseEvent
	OldEmail string `json:"old_email"`
	NewEmail string `json:"new_email"`
}

type UserDeleted struct {
	eventstore.BaseEvent
	Reason string `json:"reason"`
}

type RoleAssigned struct {
	eventstore.BaseEvent
	RoleID string `json:"role_id"`
}

type RoleUnassigned struct {
	eventstore.BaseEvent
	RoleID string `json:"role_id"`
}

func ApplyUser(s *UserState, e eventstore.Event) {
	switch ev := e.(type) {
	case *UserCreated:
		s.ID = ev.AggregateID()
		s.Email = ev.Email
		s.PassHash = ev.PassHash
		s.Active = true
	case *EmailChanged:
		s.Email = ev.NewEmail
	case *UserDeleted:
		s.Active = false
	case *RoleAssigned:
		s.RoleIDs = append(s.RoleIDs, ev.RoleID)
	case *RoleUnassigned:
		kept := s.RoleIDs[:0]
		for _, id := range s.RoleIDs {
			if id != ev.RoleID {
				kept = append(kept, id)
			}
		}
		s.RoleIDs = kept
	}
}

func NewUserAggregate(id string) *eventstore.Aggregate[UserState] {
	return eventstore.New[UserState](id, ApplyUser)
}
