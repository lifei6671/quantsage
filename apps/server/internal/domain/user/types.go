package user

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
)

func buildUser(row dbgen.AppUser) User {
	user := User{
		ID:           row.ID,
		Username:     row.Username,
		DisplayName:  row.DisplayName,
		PasswordHash: row.PasswordHash,
		Status:       row.Status,
		Role:         row.Role,
		CreatedAt:    timestampToTime(row.CreatedAt),
		UpdatedAt:    timestampToTime(row.UpdatedAt),
	}
	if row.LastLoginAt.Valid {
		loginAt := row.LastLoginAt.Time
		user.LastLoginAt = &loginAt
	}

	return user
}

func timestampValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}

func timestampToTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}
