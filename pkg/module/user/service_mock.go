package user

import (
	"context"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/seeder"
)

// MockUserService implements the local userService interface for testing.
type MockUserService struct {
	GetByIDResult          *UserReadModel
	GetByIDErr             error
	GetByEmailAuthResult   *auth.UserRecord
	GetByEmailAuthErr      error
	GetByEmailSeederResult *seeder.UserRecord
	GetByEmailSeederErr    error
	CreateUserResult       string
	CreateUserErr          error
	AssignRoleErr          error
}

func (m *MockUserService) GetByID(_ context.Context, _ string) (*UserReadModel, error) {
	return m.GetByIDResult, m.GetByIDErr
}
func (m *MockUserService) GetByEmailForAuth(_ context.Context, _ string) (*auth.UserRecord, error) {
	return m.GetByEmailAuthResult, m.GetByEmailAuthErr
}
func (m *MockUserService) GetByEmailForSeeder(_ context.Context, _ string) (*seeder.UserRecord, error) {
	return m.GetByEmailSeederResult, m.GetByEmailSeederErr
}
func (m *MockUserService) CreateUserForSeeder(_ context.Context, _ seeder.CreateUserCmd) (string, error) {
	return m.CreateUserResult, m.CreateUserErr
}
func (m *MockUserService) AssignRole(_ context.Context, _, _, _ string) error {
	return m.AssignRoleErr
}
