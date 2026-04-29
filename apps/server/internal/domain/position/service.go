package position

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

const (
	dateLayout        = "2006-01-02"
	pgUniqueViolation = "23505"
)

// Position 表示用户持仓记录。
type Position struct {
	ID           int64
	UserID       int64
	TSCode       string
	PositionDate time.Time
	Quantity     string
	CostPrice    string
	Note         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateInput 定义新增持仓参数。
type CreateInput struct {
	TSCode       string
	PositionDate string
	Quantity     string
	CostPrice    string
	Note         string
}

// UpdateInput 定义更新持仓参数。
type UpdateInput struct {
	TSCode       string
	PositionDate string
	Quantity     string
	CostPrice    string
	Note         string
}

// Service 定义用户持仓领域服务契约。
type Service interface {
	List(ctx context.Context, userID int64) ([]Position, error)
	Create(ctx context.Context, userID int64, input CreateInput) (Position, error)
	Update(ctx context.Context, userID, positionID int64, input UpdateInput) (Position, error)
	Delete(ctx context.Context, userID, positionID int64) error
}

// Querier 定义持仓领域依赖的查询接口。
type Querier interface {
	ListUserPositions(ctx context.Context, userID int64) ([]dbgen.UserPosition, error)
	CreateUserPosition(ctx context.Context, arg dbgen.CreateUserPositionParams) (dbgen.UserPosition, error)
	UpdateUserPosition(ctx context.Context, arg dbgen.UpdateUserPositionParams) (dbgen.UserPosition, error)
	DeleteUserPosition(ctx context.Context, arg dbgen.DeleteUserPositionParams) (int64, error)
}

type service struct {
	querier Querier
}

// NewService 创建用户持仓服务。
func NewService(querier Querier) Service {
	return &service{querier: querier}
}

// List 返回当前用户的全部持仓。
func (s *service) List(ctx context.Context, userID int64) ([]Position, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	rows, err := s.querier.ListUserPositions(ctx, userID)
	if err != nil {
		return nil, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("list user positions: %w", err))
	}

	items := make([]Position, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildPosition(row))
	}
	return items, nil
}

// Create 新增一条当前用户持仓记录。
func (s *service) Create(ctx context.Context, userID int64, input CreateInput) (Position, error) {
	if err := validateUserID(userID); err != nil {
		return Position{}, err
	}

	params, err := buildCreateParams(userID, input)
	if err != nil {
		return Position{}, err
	}

	row, err := s.querier.CreateUserPosition(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return Position{}, apperror.New(apperror.CodeValidationFailed, fmt.Errorf("position already exists: %w", err))
		}
		return Position{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("create user position: %w", err))
	}

	return buildPosition(row), nil
}

// Update 修改当前用户的一条持仓记录。
func (s *service) Update(ctx context.Context, userID, positionID int64, input UpdateInput) (Position, error) {
	if err := validateUserID(userID); err != nil {
		return Position{}, err
	}
	if positionID <= 0 {
		return Position{}, apperror.New(apperror.CodeBadRequest, errors.New("invalid position id"))
	}

	params, err := buildUpdateParams(userID, positionID, input)
	if err != nil {
		return Position{}, err
	}

	row, err := s.querier.UpdateUserPosition(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Position{}, apperror.New(apperror.CodeNotFound, fmt.Errorf("position %d not found: %w", positionID, err))
		}
		if isUniqueViolation(err) {
			return Position{}, apperror.New(apperror.CodeValidationFailed, fmt.Errorf("position already exists: %w", err))
		}
		return Position{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("update user position: %w", err))
	}

	return buildPosition(row), nil
}

// Delete 删除当前用户的一条持仓记录。
func (s *service) Delete(ctx context.Context, userID, positionID int64) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	if positionID <= 0 {
		return apperror.New(apperror.CodeBadRequest, errors.New("invalid position id"))
	}

	affected, err := s.querier.DeleteUserPosition(ctx, dbgen.DeleteUserPositionParams{
		ID:     positionID,
		UserID: userID,
	})
	if err != nil {
		return apperror.New(apperror.CodeDatabaseError, fmt.Errorf("delete user position: %w", err))
	}
	if affected == 0 {
		return apperror.New(apperror.CodeNotFound, fmt.Errorf("position %d not found", positionID))
	}

	return nil
}

func buildCreateParams(userID int64, input CreateInput) (dbgen.CreateUserPositionParams, error) {
	positionDate, quantity, costPrice, tsCode, note, err := normalizeInput(input.TSCode, input.PositionDate, input.Quantity, input.CostPrice, input.Note)
	if err != nil {
		return dbgen.CreateUserPositionParams{}, err
	}

	return dbgen.CreateUserPositionParams{
		UserID:       userID,
		TsCode:       tsCode,
		PositionDate: positionDate,
		Quantity:     quantity,
		CostPrice:    costPrice,
		Note:         note,
	}, nil
}

func buildUpdateParams(userID, positionID int64, input UpdateInput) (dbgen.UpdateUserPositionParams, error) {
	positionDate, quantity, costPrice, tsCode, note, err := normalizeInput(input.TSCode, input.PositionDate, input.Quantity, input.CostPrice, input.Note)
	if err != nil {
		return dbgen.UpdateUserPositionParams{}, err
	}

	return dbgen.UpdateUserPositionParams{
		ID:           positionID,
		UserID:       userID,
		TsCode:       tsCode,
		PositionDate: positionDate,
		Quantity:     quantity,
		CostPrice:    costPrice,
		Note:         note,
	}, nil
}

func normalizeInput(tsCodeText, positionDateText, quantityText, costPriceText, noteText string) (pgtype.Date, pgtype.Numeric, pgtype.Numeric, string, string, error) {
	tsCode := strings.TrimSpace(strings.ToUpper(tsCodeText))
	if tsCode == "" {
		return pgtype.Date{}, pgtype.Numeric{}, pgtype.Numeric{}, "", "", apperror.New(apperror.CodeValidationFailed, errors.New("ts_code is required"))
	}

	positionDateTime, err := time.Parse(dateLayout, strings.TrimSpace(positionDateText))
	if err != nil {
		return pgtype.Date{}, pgtype.Numeric{}, pgtype.Numeric{}, "", "", apperror.New(apperror.CodeValidationFailed, fmt.Errorf("parse position_date: %w", err))
	}
	quantity, err := parseNumeric(quantityText)
	if err != nil {
		return pgtype.Date{}, pgtype.Numeric{}, pgtype.Numeric{}, "", "", apperror.New(apperror.CodeValidationFailed, fmt.Errorf("parse quantity: %w", err))
	}
	costPrice, err := parseNumeric(costPriceText)
	if err != nil {
		return pgtype.Date{}, pgtype.Numeric{}, pgtype.Numeric{}, "", "", apperror.New(apperror.CodeValidationFailed, fmt.Errorf("parse cost_price: %w", err))
	}

	return pgtype.Date{Time: positionDateTime, Valid: true}, quantity, costPrice, tsCode, strings.TrimSpace(noteText), nil
}

func parseNumeric(value string) (pgtype.Numeric, error) {
	parsed, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return pgtype.Numeric{}, err
	}

	return pgtype.Numeric{
		Int:   parsed.Coefficient(),
		Exp:   int32(parsed.Exponent()),
		Valid: true,
	}, nil
}

func buildPosition(row dbgen.UserPosition) Position {
	return Position{
		ID:           row.ID,
		UserID:       row.UserID,
		TSCode:       row.TsCode,
		PositionDate: dateToTime(row.PositionDate),
		Quantity:     numericToString(row.Quantity),
		CostPrice:    numericToString(row.CostPrice),
		Note:         row.Note,
		CreatedAt:    timestampToTime(row.CreatedAt),
		UpdatedAt:    timestampToTime(row.UpdatedAt),
	}
}

func numericToString(value pgtype.Numeric) string {
	if !value.Valid || value.Int == nil {
		return ""
	}

	return decimal.NewFromBigInt(new(big.Int).Set(value.Int), value.Exp).String()
}

func dateToTime(value pgtype.Date) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}

func timestampToTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
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
