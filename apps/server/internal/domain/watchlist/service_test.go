package watchlist

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type fakeWatchlistQuerier struct {
	groups           []dbgen.WatchlistGroup
	items            []dbgen.WatchlistItem
	deletedGroupRows int64
	deletedItemRows  int64

	listGroupUserID   int64
	listItemsParams   dbgen.ListWatchlistItemsParams
	deleteGroupParams dbgen.DeleteWatchlistGroupParams
	deleteItemParams  dbgen.DeleteWatchlistItemParams
}

func (f *fakeWatchlistQuerier) ListWatchlistGroups(ctx context.Context, userID int64) ([]dbgen.WatchlistGroup, error) {
	f.listGroupUserID = userID
	items := make([]dbgen.WatchlistGroup, 0)
	for _, group := range f.groups {
		if group.UserID == userID {
			items = append(items, group)
		}
	}
	return items, nil
}

func (f *fakeWatchlistQuerier) GetWatchlistGroup(ctx context.Context, arg dbgen.GetWatchlistGroupParams) (dbgen.WatchlistGroup, error) {
	return dbgen.WatchlistGroup{}, nil
}

func (f *fakeWatchlistQuerier) CreateWatchlistGroup(ctx context.Context, arg dbgen.CreateWatchlistGroupParams) (dbgen.WatchlistGroup, error) {
	return dbgen.WatchlistGroup{}, nil
}

func (f *fakeWatchlistQuerier) UpdateWatchlistGroup(ctx context.Context, arg dbgen.UpdateWatchlistGroupParams) (dbgen.WatchlistGroup, error) {
	return dbgen.WatchlistGroup{}, nil
}

func (f *fakeWatchlistQuerier) DeleteWatchlistGroup(ctx context.Context, arg dbgen.DeleteWatchlistGroupParams) (int64, error) {
	f.deleteGroupParams = arg
	return f.deletedGroupRows, nil
}

func (f *fakeWatchlistQuerier) ListWatchlistItems(ctx context.Context, arg dbgen.ListWatchlistItemsParams) ([]dbgen.WatchlistItem, error) {
	f.listItemsParams = arg
	return f.items, nil
}

func (f *fakeWatchlistQuerier) CreateWatchlistItem(ctx context.Context, arg dbgen.CreateWatchlistItemParams) (dbgen.WatchlistItem, error) {
	return dbgen.WatchlistItem{}, nil
}

func (f *fakeWatchlistQuerier) DeleteWatchlistItem(ctx context.Context, arg dbgen.DeleteWatchlistItemParams) (int64, error) {
	f.deleteItemParams = arg
	return f.deletedItemRows, nil
}

func TestListGroupsOnlyReturnsCurrentUserGroups(t *testing.T) {
	t.Parallel()

	createdAt := timestampValue(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC))
	querier := &fakeWatchlistQuerier{
		groups: []dbgen.WatchlistGroup{
			{ID: 1, UserID: 10, Name: "我的自选", CreatedAt: createdAt, UpdatedAt: createdAt},
			{ID: 2, UserID: 20, Name: "其他用户", CreatedAt: createdAt, UpdatedAt: createdAt},
		},
	}
	svc := NewService(querier)

	groups, err := svc.ListGroups(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}
	if querier.listGroupUserID != 10 {
		t.Fatalf("ListGroups userID = %d, want %d", querier.listGroupUserID, 10)
	}
	if len(groups) != 1 || groups[0].ID != 1 {
		t.Fatalf("groups = %+v, want only current user's group", groups)
	}
}

func TestListItemsUsesGroupAndCurrentUserScope(t *testing.T) {
	t.Parallel()

	querier := &fakeWatchlistQuerier{
		items: []dbgen.WatchlistItem{{
			ID:        7,
			GroupID:   3,
			TsCode:    "000001.SZ",
			CreatedAt: timestampValue(time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)),
		}},
	}
	svc := NewService(querier)

	items, err := svc.ListItems(context.Background(), 10, 3)
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}
	if querier.listItemsParams.UserID != 10 || querier.listItemsParams.GroupID != 3 {
		t.Fatalf("ListItems params = %+v, want user_id=10 group_id=3", querier.listItemsParams)
	}
	if len(items) != 1 || items[0].ID != 7 {
		t.Fatalf("items = %+v, want current scoped item", items)
	}
}

func TestDeleteGroupRejectsOtherUserGroup(t *testing.T) {
	t.Parallel()

	querier := &fakeWatchlistQuerier{deletedGroupRows: 0}
	svc := NewService(querier)

	err := svc.DeleteGroup(context.Background(), 10, 88)
	if err == nil {
		t.Fatal("DeleteGroup() error = nil, want non-nil")
	}
	if querier.deleteGroupParams.UserID != 10 || querier.deleteGroupParams.ID != 88 {
		t.Fatalf("DeleteGroup params = %+v, want user_id=10 id=88", querier.deleteGroupParams)
	}
	if got := apperror.CodeOf(err); got != apperror.CodeNotFound {
		t.Fatalf("CodeOf(err) = %d, want %d", got, apperror.CodeNotFound)
	}
}

func TestDeleteItemRejectsOtherUserItem(t *testing.T) {
	t.Parallel()

	querier := &fakeWatchlistQuerier{deletedItemRows: 0}
	svc := NewService(querier)

	err := svc.DeleteItem(context.Background(), 10, 3, 99)
	if err == nil {
		t.Fatal("DeleteItem() error = nil, want non-nil")
	}
	if querier.deleteItemParams.UserID != 10 || querier.deleteItemParams.GroupID != 3 || querier.deleteItemParams.ID != 99 {
		t.Fatalf("DeleteItem params = %+v, want user_id=10 group_id=3 id=99", querier.deleteItemParams)
	}
	if got := apperror.CodeOf(err); got != apperror.CodeNotFound {
		t.Fatalf("CodeOf(err) = %d, want %d", got, apperror.CodeNotFound)
	}
}

func timestampValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
