package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

var errDuplicateBootstrapUsername = errors.New("duplicate bootstrap username")

// User 表示系统中的一个登录账号。
type User struct {
	ID           int64
	Username     string
	DisplayName  string
	PasswordHash string
	Status       string
	Role         string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// BootstrapUser 表示需要在启动阶段同步的预置账号。
type BootstrapUser struct {
	Username     string
	DisplayName  string
	PasswordHash string
	Status       string
	Role         string
}

// Service 定义用户领域服务契约。
type Service interface {
	GetByID(ctx context.Context, userID int64) (User, error)
	Authenticate(ctx context.Context, username, password string) (User, error)
	SyncBootstrapUsers(ctx context.Context, users []BootstrapUser) error
}

// Querier 定义用户领域依赖的数据库查询接口。
type Querier interface {
	GetUserByID(ctx context.Context, id int64) (dbgen.AppUser, error)
	GetUserByUsername(ctx context.Context, username string) (dbgen.AppUser, error)
	CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.AppUser, error)
	UpdateBootstrapUser(ctx context.Context, arg dbgen.UpdateBootstrapUserParams) (dbgen.AppUser, error)
	TouchUserLastLogin(ctx context.Context, arg dbgen.TouchUserLastLoginParams) error
}

type service struct {
	querier Querier
	now     func() time.Time
}

// NewService 创建用户领域服务。
func NewService(querier Querier, now func() time.Time) Service {
	if now == nil {
		now = time.Now
	}

	return &service{
		querier: querier,
		now:     now,
	}
}

// GetByID 按用户 ID 查询当前账号。
func (s *service) GetByID(ctx context.Context, userID int64) (User, error) {
	if s.querier == nil {
		return User{}, apperror.New(apperror.CodeInternal, errors.New("user service is not configured"))
	}
	if userID <= 0 {
		return User{}, apperror.New(apperror.CodeUnauthorized, errors.New("invalid user id"))
	}

	row, err := s.querier.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, apperror.New(apperror.CodeUnauthorized, fmt.Errorf("user %d not found: %w", userID, err))
		}
		return User{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("get user by id: %w", err))
	}
	// 会话恢复也必须复用账号状态校验，避免已禁用账号继续持有旧 session 访问私有接口。
	if !strings.EqualFold(row.Status, "active") {
		return User{}, apperror.New(apperror.CodeUnauthorized, errors.New("user is disabled"))
	}

	return buildUser(row), nil
}

// Authenticate 校验用户名和密码，并在成功后刷新最近登录时间。
func (s *service) Authenticate(ctx context.Context, username, password string) (User, error) {
	if s.querier == nil {
		return User{}, apperror.New(apperror.CodeInternal, errors.New("user service is not configured"))
	}

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return User{}, apperror.New(apperror.CodeUnauthorized, errors.New("username or password is empty"))
	}

	row, err := s.querier.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, apperror.New(apperror.CodeUnauthorized, fmt.Errorf("user %s not found: %w", username, err))
		}
		return User{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("get user by username: %w", err))
	}
	if !strings.EqualFold(row.Status, "active") {
		return User{}, apperror.New(apperror.CodeUnauthorized, errors.New("user is disabled"))
	}
	if err := bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)); err != nil {
		return User{}, apperror.New(apperror.CodeUnauthorized, fmt.Errorf("password mismatch: %w", err))
	}

	loginAt := s.now().UTC()
	if err := s.querier.TouchUserLastLogin(ctx, dbgen.TouchUserLastLoginParams{
		ID:          row.ID,
		LastLoginAt: timestampValue(loginAt),
	}); err != nil {
		return User{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("touch user last login: %w", err))
	}

	user := buildUser(row)
	user.LastLoginAt = &loginAt
	return user, nil
}

// SyncBootstrapUsers 幂等同步配置文件中的预置账号。
func (s *service) SyncBootstrapUsers(ctx context.Context, users []BootstrapUser) error {
	if s.querier == nil {
		return apperror.New(apperror.CodeInternal, errors.New("user service is not configured"))
	}
	if err := validateBootstrapUsers(users); err != nil {
		return apperror.New(apperror.CodeValidationFailed, fmt.Errorf("validate bootstrap users: %w", err))
	}

	for _, item := range users {
		row, err := s.querier.GetUserByUsername(ctx, item.Username)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return apperror.New(apperror.CodeDatabaseError, fmt.Errorf("get bootstrap user %s: %w", item.Username, err))
		}

		if errors.Is(err, pgx.ErrNoRows) {
			if _, createErr := s.querier.CreateUser(ctx, dbgen.CreateUserParams{
				Username:     item.Username,
				DisplayName:  item.DisplayName,
				PasswordHash: item.PasswordHash,
				Status:       item.Status,
				Role:         item.Role,
			}); createErr != nil {
				return apperror.New(apperror.CodeDatabaseError, fmt.Errorf("create bootstrap user %s: %w", item.Username, createErr))
			}
			continue
		}

		if _, updateErr := s.querier.UpdateBootstrapUser(ctx, dbgen.UpdateBootstrapUserParams{
			Username:     row.Username,
			DisplayName:  item.DisplayName,
			PasswordHash: item.PasswordHash,
			Status:       item.Status,
			Role:         item.Role,
		}); updateErr != nil {
			return apperror.New(apperror.CodeDatabaseError, fmt.Errorf("update bootstrap user %s: %w", item.Username, updateErr))
		}
	}

	return nil
}

func validateBootstrapUsers(users []BootstrapUser) error {
	seen := make(map[string]struct{}, len(users))
	for _, item := range users {
		username := strings.TrimSpace(item.Username)
		if username == "" {
			return errors.New("bootstrap username is required")
		}
		if _, ok := seen[username]; ok {
			return fmt.Errorf("%w: %s", errDuplicateBootstrapUsername, username)
		}
		seen[username] = struct{}{}
		if strings.TrimSpace(item.DisplayName) == "" {
			return fmt.Errorf("bootstrap user %s display_name is required", username)
		}
		if strings.TrimSpace(item.PasswordHash) == "" {
			return fmt.Errorf("bootstrap user %s password_hash is required", username)
		}
	}

	return nil
}
