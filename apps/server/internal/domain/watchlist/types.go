package watchlist

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
)

func buildGroup(row dbgen.WatchlistGroup) Group {
	return Group{
		ID:        row.ID,
		UserID:    row.UserID,
		Name:      row.Name,
		SortOrder: row.SortOrder,
		CreatedAt: timestampToTime(row.CreatedAt),
		UpdatedAt: timestampToTime(row.UpdatedAt),
	}
}

func timestampToTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}
