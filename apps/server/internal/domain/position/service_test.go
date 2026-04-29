package position

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type fakePositionQuerier struct {
	positions        []dbgen.UserPosition
	deletedRows      int64
	listUserID       int64
	deleteUserParams dbgen.DeleteUserPositionParams
	createUserParams dbgen.CreateUserPositionParams
	updateUserParams dbgen.UpdateUserPositionParams
}

func (f *fakePositionQuerier) ListUserPositions(ctx context.Context, userID int64) ([]dbgen.UserPosition, error) {
	f.listUserID = userID
	items := make([]dbgen.UserPosition, 0)
	for _, position := range f.positions {
		if position.UserID == userID {
			items = append(items, position)
		}
	}
	return items, nil
}

func (f *fakePositionQuerier) CreateUserPosition(ctx context.Context, arg dbgen.CreateUserPositionParams) (dbgen.UserPosition, error) {
	f.createUserParams = arg
	return dbgen.UserPosition{
		ID:           1,
		UserID:       arg.UserID,
		TsCode:       arg.TsCode,
		PositionDate: arg.PositionDate,
		Quantity:     arg.Quantity,
		CostPrice:    arg.CostPrice,
		Note:         arg.Note,
		CreatedAt:    timestampValue(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)),
		UpdatedAt:    timestampValue(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)),
	}, nil
}

func (f *fakePositionQuerier) UpdateUserPosition(ctx context.Context, arg dbgen.UpdateUserPositionParams) (dbgen.UserPosition, error) {
	f.updateUserParams = arg
	return dbgen.UserPosition{}, nil
}

func (f *fakePositionQuerier) DeleteUserPosition(ctx context.Context, arg dbgen.DeleteUserPositionParams) (int64, error) {
	f.deleteUserParams = arg
	return f.deletedRows, nil
}

func TestListPositionsOnlyReturnsCurrentUserPositions(t *testing.T) {
	t.Parallel()

	positionDate := pgtype.Date{Time: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC), Valid: true}
	createdAt := timestampValue(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	querier := &fakePositionQuerier{
		positions: []dbgen.UserPosition{
			{
				ID:           1,
				UserID:       10,
				TsCode:       "000001.SZ",
				PositionDate: positionDate,
				Quantity:     numericValue(100),
				CostPrice:    numericValue(12),
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			},
			{
				ID:           2,
				UserID:       20,
				TsCode:       "600519.SH",
				PositionDate: positionDate,
				Quantity:     numericValue(1),
				CostPrice:    numericValue(1500),
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			},
		},
	}
	svc := NewService(querier)

	positions, err := svc.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if querier.listUserID != 10 {
		t.Fatalf("List userID = %d, want %d", querier.listUserID, 10)
	}
	if len(positions) != 1 || positions[0].ID != 1 {
		t.Fatalf("positions = %+v, want only current user's position", positions)
	}
}

func TestCreatePositionUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	querier := &fakePositionQuerier{}
	svc := NewService(querier)

	position, err := svc.Create(context.Background(), 10, CreateInput{
		TSCode:       "000001.sz",
		PositionDate: "2026-04-28",
		Quantity:     "100",
		CostPrice:    "12.34",
		Note:         "  test note  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if querier.createUserParams.UserID != 10 || querier.createUserParams.TsCode != "000001.SZ" {
		t.Fatalf("Create params = %+v, want current user and upper ts_code", querier.createUserParams)
	}
	if position.UserID != 10 || position.TSCode != "000001.SZ" || position.Note != "test note" {
		t.Fatalf("position = %+v, want normalized current user's position", position)
	}
}

func TestDeletePositionRejectsOtherUserPosition(t *testing.T) {
	t.Parallel()

	querier := &fakePositionQuerier{deletedRows: 0}
	svc := NewService(querier)

	err := svc.Delete(context.Background(), 10, 88)
	if err == nil {
		t.Fatal("Delete() error = nil, want non-nil")
	}
	if querier.deleteUserParams.UserID != 10 || querier.deleteUserParams.ID != 88 {
		t.Fatalf("Delete params = %+v, want user_id=10 id=88", querier.deleteUserParams)
	}
	if got := apperror.CodeOf(err); got != apperror.CodeNotFound {
		t.Fatalf("CodeOf(err) = %d, want %d", got, apperror.CodeNotFound)
	}
}

func timestampValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func numericValue(value int64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(value), Exp: 0, Valid: true}
}
