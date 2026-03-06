package user_test

import (
	"slices"
	"testing"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/user"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

// ---

func newBase(_ string) eventstore.BaseEvent {
	return eventstore.BaseEvent{ /* set AggregateID field, or use your constructor */ }
}

// helpers to build events concisely

func userCreated(id, email, pass string) *user.UserCreated {
	e := &user.UserCreated{Email: email, PassHash: pass}
	// set aggregate ID via whatever mechanism your BaseEvent exposes, e.g.:
	e.BaseEvent = newBase(id)
	return e
}

func emailChanged(oldEmail, newEmail string) *user.EmailChanged {
	return &user.EmailChanged{OldEmail: oldEmail, NewEmail: newEmail}
}

func userDeleted(reason string) *user.UserDeleted   { return &user.UserDeleted{Reason: reason} }
func roleAssigned(id string) *user.RoleAssigned     { return &user.RoleAssigned{RoleID: id} }
func roleUnassigned(id string) *user.RoleUnassigned { return &user.RoleUnassigned{RoleID: id} }

// ---

func TestApplyUser_UserCreated(t *testing.T) {
	s := &user.UserState{}
	user.ApplyUser(s, userCreated("u1", "alice@example.com", "hash123"))

	if s.Email != "alice@example.com" {
		t.Errorf("Email: got %q, want %q", s.Email, "alice@example.com")
	}
	if s.PassHash != "hash123" {
		t.Errorf("PassHash: got %q, want %q", s.PassHash, "hash123")
	}
	if !s.Active {
		t.Error("Active: want true after creation")
	}
}

func TestApplyUser_EmailChanged(t *testing.T) {
	s := &user.UserState{Email: "old@example.com"}
	user.ApplyUser(s, emailChanged("old@example.com", "new@example.com"))

	if s.Email != "new@example.com" {
		t.Errorf("Email: got %q, want %q", s.Email, "new@example.com")
	}
}

func TestApplyUser_UserDeleted(t *testing.T) {
	s := &user.UserState{Active: true}
	user.ApplyUser(s, userDeleted("spam"))

	if s.Active {
		t.Error("Active: want false after deletion")
	}
}

func TestApplyUser_RoleAssigned(t *testing.T) {
	s := &user.UserState{}
	user.ApplyUser(s, roleAssigned("admin"))
	user.ApplyUser(s, roleAssigned("editor"))

	want := []string{"admin", "editor"}
	if !slices.Equal(s.RoleIDs, want) {
		t.Errorf("RoleIDs: got %v, want %v", s.RoleIDs, want)
	}
}

func TestApplyUser_RoleUnassigned_RemovesCorrectRole(t *testing.T) {
	s := &user.UserState{RoleIDs: []string{"admin", "editor", "viewer"}}
	user.ApplyUser(s, roleUnassigned("editor"))

	want := []string{"admin", "viewer"}
	if !slices.Equal(s.RoleIDs, want) {
		t.Errorf("RoleIDs: got %v, want %v", s.RoleIDs, want)
	}
}

func TestApplyUser_RoleUnassigned_NonExistentRoleIsNoop(t *testing.T) {
	s := &user.UserState{RoleIDs: []string{"admin"}}
	user.ApplyUser(s, roleUnassigned("ghost"))

	if !slices.Equal(s.RoleIDs, []string{"admin"}) {
		t.Errorf("RoleIDs: got %v, want [admin]", s.RoleIDs)
	}
}

func TestApplyUser_RoleUnassigned_LastRole(t *testing.T) {
	s := &user.UserState{RoleIDs: []string{"admin"}}
	user.ApplyUser(s, roleUnassigned("admin"))

	if len(s.RoleIDs) != 0 {
		t.Errorf("RoleIDs: got %v, want empty", s.RoleIDs)
	}
}

func TestApplyUser_FullLifecycle(t *testing.T) {
	s := &user.UserState{}

	user.ApplyUser(s, userCreated("u42", "alice@example.com", "phash"))
	user.ApplyUser(s, roleAssigned("admin"))
	user.ApplyUser(s, emailChanged("alice@example.com", "alice2@example.com"))
	user.ApplyUser(s, roleAssigned("editor"))
	user.ApplyUser(s, roleUnassigned("admin"))
	user.ApplyUser(s, userDeleted("account closed"))

	if s.Email != "alice2@example.com" {
		t.Errorf("Email: got %q", s.Email)
	}
	if s.Active {
		t.Error("Active: want false")
	}
	if !slices.Equal(s.RoleIDs, []string{"editor"}) {
		t.Errorf("RoleIDs: got %v, want [editor]", s.RoleIDs)
	}
}

func TestApplyUser_UnknownEventIsNoop(t *testing.T) {
	type unknownEvent struct{ eventstore.BaseEvent }

	s := &user.UserState{Email: "x@example.com", Active: true}
	user.ApplyUser(s, &unknownEvent{})

	if s.Email != "x@example.com" || !s.Active {
		t.Error("unknown event mutated state unexpectedly")
	}
}
