package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type fakeQuerier struct {
	users         map[string]dbgen.AppUser
	touchedUserID int64
	createCalls   int
	updateCalls   int
	createUserErr error
	updateUserErr error
	touchLoginErr error
	getByUserErr  error
	getByIDErr    error
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{
		users: make(map[string]dbgen.AppUser),
	}
}

func (f *fakeQuerier) GetUserByID(ctx context.Context, id int64) (dbgen.AppUser, error) {
	if f.getByIDErr != nil {
		return dbgen.AppUser{}, f.getByIDErr
	}
	for _, item := range f.users {
		if item.ID == id {
			return item, nil
		}
	}
	return dbgen.AppUser{}, pgx.ErrNoRows
}

func (f *fakeQuerier) GetUserByUsername(ctx context.Context, username string) (dbgen.AppUser, error) {
	if f.getByUserErr != nil {
		return dbgen.AppUser{}, f.getByUserErr
	}
	item, ok := f.users[username]
	if !ok {
		return dbgen.AppUser{}, pgx.ErrNoRows
	}
	return item, nil
}

func (f *fakeQuerier) CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.AppUser, error) {
	if f.createUserErr != nil {
		return dbgen.AppUser{}, f.createUserErr
	}
	f.createCalls++
	item := dbgen.AppUser{
		ID:           int64(len(f.users) + 1),
		Username:     arg.Username,
		DisplayName:  arg.DisplayName,
		PasswordHash: arg.PasswordHash,
		Status:       arg.Status,
		Role:         arg.Role,
	}
	f.users[arg.Username] = item
	return item, nil
}

func (f *fakeQuerier) UpdateBootstrapUser(ctx context.Context, arg dbgen.UpdateBootstrapUserParams) (dbgen.AppUser, error) {
	if f.updateUserErr != nil {
		return dbgen.AppUser{}, f.updateUserErr
	}
	f.updateCalls++
	item := f.users[arg.Username]
	item.DisplayName = arg.DisplayName
	item.PasswordHash = arg.PasswordHash
	item.Status = arg.Status
	item.Role = arg.Role
	f.users[arg.Username] = item
	return item, nil
}

func (f *fakeQuerier) TouchUserLastLogin(ctx context.Context, arg dbgen.TouchUserLastLoginParams) error {
	if f.touchLoginErr != nil {
		return f.touchLoginErr
	}
	f.touchedUserID = arg.ID
	for key, item := range f.users {
		if item.ID == arg.ID {
			item.LastLoginAt = arg.LastLoginAt
			f.users[key] = item
			return nil
		}
	}
	return nil
}

func TestSyncBootstrapUsersIsIdempotent(t *testing.T) {
	t.Parallel()

	querier := newFakeQuerier()
	svc := NewService(querier, func() time.Time { return time.Unix(10, 0).UTC() })
	users := []BootstrapUser{{
		Username:     "admin",
		DisplayName:  "管理员",
		PasswordHash: "hash-1",
		Status:       "active",
		Role:         "admin",
	}}

	if err := svc.SyncBootstrapUsers(context.Background(), users); err != nil {
		t.Fatalf("SyncBootstrapUsers() first call error = %v", err)
	}
	if err := svc.SyncBootstrapUsers(context.Background(), []BootstrapUser{{
		Username:     "admin",
		DisplayName:  "管理员-已同步",
		PasswordHash: "hash-2",
		Status:       "active",
		Role:         "admin",
	}}); err != nil {
		t.Fatalf("SyncBootstrapUsers() second call error = %v", err)
	}

	if querier.createCalls != 1 {
		t.Fatalf("querier.createCalls = %d, want %d", querier.createCalls, 1)
	}
	if querier.updateCalls != 1 {
		t.Fatalf("querier.updateCalls = %d, want %d", querier.updateCalls, 1)
	}
	if querier.users["admin"].DisplayName != "管理员-已同步" {
		t.Fatalf("display name = %q, want %q", querier.users["admin"].DisplayName, "管理员-已同步")
	}
}

func TestSyncBootstrapUsersRejectsDuplicateUsernames(t *testing.T) {
	t.Parallel()

	svc := NewService(newFakeQuerier(), time.Now)
	err := svc.SyncBootstrapUsers(context.Background(), []BootstrapUser{
		{Username: "admin", DisplayName: "管理员", PasswordHash: "hash-1"},
		{Username: "admin", DisplayName: "管理员2", PasswordHash: "hash-2"},
	})
	if err == nil {
		t.Fatal("SyncBootstrapUsers() error = nil, want non-nil")
	}
	if got := apperror.CodeOf(err); got != apperror.CodeValidationFailed {
		t.Fatalf("CodeOf(err) = %d, want %d", got, apperror.CodeValidationFailed)
	}
	if !errors.Is(err, errDuplicateBootstrapUsername) {
		t.Fatalf("errors.Is(err, errDuplicateBootstrapUsername) = false, want true")
	}
}

func TestAuthenticateTouchesLastLogin(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("demo-123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	querier := newFakeQuerier()
	querier.users["admin"] = dbgen.AppUser{
		ID:           7,
		Username:     "admin",
		DisplayName:  "管理员",
		PasswordHash: string(hash),
		Status:       "active",
		Role:         "admin",
	}
	now := time.Date(2026, 4, 28, 9, 30, 0, 0, time.UTC)
	svc := NewService(querier, func() time.Time { return now })

	user, authErr := svc.Authenticate(context.Background(), "admin", "demo-123")
	if authErr != nil {
		t.Fatalf("Authenticate() error = %v", authErr)
	}
	if querier.touchedUserID != 7 {
		t.Fatalf("querier.touchedUserID = %d, want %d", querier.touchedUserID, 7)
	}
	if user.LastLoginAt == nil || !user.LastLoginAt.Equal(now) {
		t.Fatalf("user.LastLoginAt = %v, want %v", user.LastLoginAt, now)
	}
}

func TestGetByIDRejectsInactiveUser(t *testing.T) {
	t.Parallel()

	querier := newFakeQuerier()
	querier.users["admin"] = dbgen.AppUser{
		ID:           7,
		Username:     "admin",
		DisplayName:  "管理员",
		PasswordHash: "hash",
		Status:       "inactive",
		Role:         "admin",
	}
	svc := NewService(querier, time.Now)

	_, err := svc.GetByID(context.Background(), 7)
	if err == nil {
		t.Fatal("GetByID() error = nil, want non-nil")
	}
	if got := apperror.CodeOf(err); got != apperror.CodeUnauthorized {
		t.Fatalf("CodeOf(err) = %d, want %d", got, apperror.CodeUnauthorized)
	}
}
