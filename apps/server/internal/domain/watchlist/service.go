package watchlist

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

const pgUniqueViolation = "23505"

// Group 表示一个用户自选分组。
type Group struct {
	ID        int64
	UserID    int64
	Name      string
	SortOrder int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Item 表示自选分组内的一条股票记录。
type Item struct {
	ID        int64
	GroupID   int64
	TSCode    string
	Note      string
	CreatedAt time.Time
}

// CreateGroupInput 定义创建分组参数。
type CreateGroupInput struct {
	Name      string
	SortOrder int32
}

// UpdateGroupInput 定义更新分组参数。
type UpdateGroupInput struct {
	Name      string
	SortOrder int32
}

// CreateItemInput 定义新增分组内股票参数。
type CreateItemInput struct {
	TSCode string
	Note   string
}

// Service 定义用户自选股领域服务契约。
type Service interface {
	ListGroups(ctx context.Context, userID int64) ([]Group, error)
	CreateGroup(ctx context.Context, userID int64, input CreateGroupInput) (Group, error)
	UpdateGroup(ctx context.Context, userID, groupID int64, input UpdateGroupInput) (Group, error)
	DeleteGroup(ctx context.Context, userID, groupID int64) error
	ListItems(ctx context.Context, userID, groupID int64) ([]Item, error)
	CreateItem(ctx context.Context, userID, groupID int64, input CreateItemInput) (Item, error)
	DeleteItem(ctx context.Context, userID, groupID, itemID int64) error
}

// Querier 定义自选股领域依赖的查询接口。
type Querier interface {
	ListWatchlistGroups(ctx context.Context, userID int64) ([]dbgen.WatchlistGroup, error)
	GetWatchlistGroup(ctx context.Context, arg dbgen.GetWatchlistGroupParams) (dbgen.WatchlistGroup, error)
	CreateWatchlistGroup(ctx context.Context, arg dbgen.CreateWatchlistGroupParams) (dbgen.WatchlistGroup, error)
	UpdateWatchlistGroup(ctx context.Context, arg dbgen.UpdateWatchlistGroupParams) (dbgen.WatchlistGroup, error)
	DeleteWatchlistGroup(ctx context.Context, arg dbgen.DeleteWatchlistGroupParams) (int64, error)
	ListWatchlistItems(ctx context.Context, arg dbgen.ListWatchlistItemsParams) ([]dbgen.WatchlistItem, error)
	CreateWatchlistItem(ctx context.Context, arg dbgen.CreateWatchlistItemParams) (dbgen.WatchlistItem, error)
	DeleteWatchlistItem(ctx context.Context, arg dbgen.DeleteWatchlistItemParams) (int64, error)
}

type service struct {
	querier Querier
}

// NewService 创建用户自选股服务。
func NewService(querier Querier) Service {
	return &service{querier: querier}
}

// ListGroups 返回当前用户的全部自选分组。
func (s *service) ListGroups(ctx context.Context, userID int64) ([]Group, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	rows, err := s.querier.ListWatchlistGroups(ctx, userID)
	if err != nil {
		return nil, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("list watchlist groups: %w", err))
	}

	items := make([]Group, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildGroup(row))
	}
	return items, nil
}

// CreateGroup 为当前用户新增一个自选分组。
func (s *service) CreateGroup(ctx context.Context, userID int64, input CreateGroupInput) (Group, error) {
	if err := validateUserID(userID); err != nil {
		return Group{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Group{}, apperror.New(apperror.CodeValidationFailed, errors.New("watchlist group name is required"))
	}

	row, err := s.querier.CreateWatchlistGroup(ctx, dbgen.CreateWatchlistGroupParams{
		UserID:    userID,
		Name:      name,
		SortOrder: input.SortOrder,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Group{}, apperror.New(apperror.CodeValidationFailed, fmt.Errorf("watchlist group %s already exists: %w", name, err))
		}
		return Group{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("create watchlist group: %w", err))
	}

	return buildGroup(row), nil
}

// UpdateGroup 修改当前用户的自选分组。
func (s *service) UpdateGroup(ctx context.Context, userID, groupID int64, input UpdateGroupInput) (Group, error) {
	if err := validateUserID(userID); err != nil {
		return Group{}, err
	}
	if groupID <= 0 {
		return Group{}, apperror.New(apperror.CodeBadRequest, errors.New("invalid watchlist group id"))
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Group{}, apperror.New(apperror.CodeValidationFailed, errors.New("watchlist group name is required"))
	}

	row, err := s.querier.UpdateWatchlistGroup(ctx, dbgen.UpdateWatchlistGroupParams{
		ID:        groupID,
		UserID:    userID,
		Name:      name,
		SortOrder: input.SortOrder,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Group{}, apperror.New(apperror.CodeNotFound, fmt.Errorf("watchlist group %d not found: %w", groupID, err))
		}
		if isUniqueViolation(err) {
			return Group{}, apperror.New(apperror.CodeValidationFailed, fmt.Errorf("watchlist group %s already exists: %w", name, err))
		}
		return Group{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("update watchlist group: %w", err))
	}

	return buildGroup(row), nil
}

// DeleteGroup 删除当前用户的自选分组。
func (s *service) DeleteGroup(ctx context.Context, userID, groupID int64) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	if groupID <= 0 {
		return apperror.New(apperror.CodeBadRequest, errors.New("invalid watchlist group id"))
	}

	affected, err := s.querier.DeleteWatchlistGroup(ctx, dbgen.DeleteWatchlistGroupParams{
		ID:     groupID,
		UserID: userID,
	})
	if err != nil {
		return apperror.New(apperror.CodeDatabaseError, fmt.Errorf("delete watchlist group: %w", err))
	}
	if affected == 0 {
		return apperror.New(apperror.CodeNotFound, fmt.Errorf("watchlist group %d not found", groupID))
	}

	return nil
}

// ListItems 返回指定分组下的股票列表。
func (s *service) ListItems(ctx context.Context, userID, groupID int64) ([]Item, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if groupID <= 0 {
		return nil, apperror.New(apperror.CodeBadRequest, errors.New("invalid watchlist group id"))
	}

	rows, err := s.querier.ListWatchlistItems(ctx, dbgen.ListWatchlistItemsParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return nil, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("list watchlist items: %w", err))
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, Item{
			ID:        row.ID,
			GroupID:   row.GroupID,
			TSCode:    row.TsCode,
			Note:      row.Note,
			CreatedAt: timestampToTime(row.CreatedAt),
		})
	}
	return items, nil
}

// CreateItem 在当前用户的指定分组下新增股票。
func (s *service) CreateItem(ctx context.Context, userID, groupID int64, input CreateItemInput) (Item, error) {
	if err := validateUserID(userID); err != nil {
		return Item{}, err
	}
	if groupID <= 0 {
		return Item{}, apperror.New(apperror.CodeBadRequest, errors.New("invalid watchlist group id"))
	}
	tsCode := strings.TrimSpace(strings.ToUpper(input.TSCode))
	if tsCode == "" {
		return Item{}, apperror.New(apperror.CodeValidationFailed, errors.New("ts_code is required"))
	}

	row, err := s.querier.CreateWatchlistItem(ctx, dbgen.CreateWatchlistItemParams{
		ID:     groupID,
		TsCode: tsCode,
		Note:   strings.TrimSpace(input.Note),
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, apperror.New(apperror.CodeNotFound, fmt.Errorf("watchlist group %d not found: %w", groupID, err))
		}
		if isUniqueViolation(err) {
			return Item{}, apperror.New(apperror.CodeValidationFailed, fmt.Errorf("watchlist item %s already exists: %w", tsCode, err))
		}
		return Item{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("create watchlist item: %w", err))
	}

	return Item{
		ID:        row.ID,
		GroupID:   row.GroupID,
		TSCode:    row.TsCode,
		Note:      row.Note,
		CreatedAt: timestampToTime(row.CreatedAt),
	}, nil
}

// DeleteItem 删除当前用户分组内的一条股票记录。
func (s *service) DeleteItem(ctx context.Context, userID, groupID, itemID int64) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	if groupID <= 0 || itemID <= 0 {
		return apperror.New(apperror.CodeBadRequest, errors.New("invalid watchlist item id"))
	}

	affected, err := s.querier.DeleteWatchlistItem(ctx, dbgen.DeleteWatchlistItemParams{
		ID:      itemID,
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return apperror.New(apperror.CodeDatabaseError, fmt.Errorf("delete watchlist item: %w", err))
	}
	if affected == 0 {
		return apperror.New(apperror.CodeNotFound, fmt.Errorf("watchlist item %d not found", itemID))
	}

	return nil
}

func validateUserID(userID int64) error {
	if userID > 0 {
		return nil
	}

	return apperror.New(apperror.CodeUnauthorized, errors.New("invalid current user"))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr != nil && pgErr.Code == pgUniqueViolation
}
