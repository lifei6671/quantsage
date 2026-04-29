package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	redisstore "github.com/gin-contrib/sessions/redis"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lifei6671/quantsage/apps/server/internal/config"
	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
	positiondomain "github.com/lifei6671/quantsage/apps/server/internal/domain/position"
	stockdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/stock"
	strategydomain "github.com/lifei6671/quantsage/apps/server/internal/domain/strategy"
	userdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/user"
	watchlistdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/watchlist"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/db/dbgen"
)

const (
	sessionPoolSize = 10
)

// ServerRuntime 组合了本地共享底座和用户私有数据能力。
type ServerRuntime struct {
	dbPool           *pgxpool.Pool
	sampleRuntime    *SampleRuntime
	userService      userdomain.Service
	watchlistService watchlistdomain.Service
	positionService  positiondomain.Service
}

// NewServerRuntime 创建 QuantSage Server 的完整运行时。
func NewServerRuntime(ctx context.Context, cfg *config.Config) (*ServerRuntime, error) {
	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return nil, fmt.Errorf("database dsn is required")
	}

	dbPool, err := pgxpool.New(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	queries := dbgen.New(dbPool)
	userService := userdomain.NewService(queries, time.Now)
	if err := userService.SyncBootstrapUsers(ctx, buildBootstrapUsers(cfg.Auth.BootstrapUsers)); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("sync bootstrap users: %w", err)
	}

	runtime := &ServerRuntime{
		dbPool:           dbPool,
		userService:      userService,
		watchlistService: watchlistdomain.NewService(queries),
		positionService:  positiondomain.NewService(queries),
	}

	if strings.EqualFold(cfg.App.Env, "local") {
		sampleRuntime, err := NewSampleRuntime("apps/server/testdata/sample")
		if err != nil {
			dbPool.Close()
			return nil, fmt.Errorf("bootstrap sample runtime: %w", err)
		}
		runtime.sampleRuntime = sampleRuntime
	}

	return runtime, nil
}

// Close 释放运行时持有的基础设施连接。
func (r *ServerRuntime) Close() {
	if r.dbPool != nil {
		r.dbPool.Close()
	}
}

// NewSessionStore 创建 Redis Session Store。
func NewSessionStore(cfg *config.Config) (sessions.Store, error) {
	if strings.TrimSpace(cfg.Redis.Addr) == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	if strings.TrimSpace(cfg.Auth.SessionSecret) == "" {
		return nil, fmt.Errorf("auth session secret is required")
	}

	store, err := redisstore.NewStoreWithDB(
		sessionPoolSize,
		"tcp",
		cfg.Redis.Addr,
		"",
		cfg.Redis.Password,
		strconv.Itoa(cfg.Redis.DB),
		[]byte(cfg.Auth.SessionSecret),
	)
	if err != nil {
		return nil, fmt.Errorf("create redis session store: %w", err)
	}
	options, err := buildSessionOptions(cfg)
	if err != nil {
		return nil, err
	}
	store.Options(options)

	return store, nil
}

// StockService 返回共享股票查询服务。
func (r *ServerRuntime) StockService() stockdomain.Service {
	if r.sampleRuntime == nil {
		return nil
	}

	return r.sampleRuntime.StockService()
}

// Runner 返回共享任务执行器。
func (r *ServerRuntime) Runner() jobdomain.Runner {
	if r.sampleRuntime == nil {
		return nil
	}

	return r.sampleRuntime.Runner()
}

// SignalQueryService 返回共享策略信号查询服务。
func (r *ServerRuntime) SignalQueryService() strategydomain.QueryService {
	if r.sampleRuntime == nil {
		return nil
	}

	return r.sampleRuntime.SignalQueryService()
}

// JobQueryService 返回共享任务记录查询服务。
func (r *ServerRuntime) JobQueryService() jobdomain.QueryService {
	if r.sampleRuntime == nil {
		return nil
	}

	return r.sampleRuntime.JobQueryService()
}

// UserService 返回用户领域服务。
func (r *ServerRuntime) UserService() userdomain.Service {
	return r.userService
}

// WatchlistService 返回用户自选股服务。
func (r *ServerRuntime) WatchlistService() watchlistdomain.Service {
	return r.watchlistService
}

// PositionService 返回用户持仓服务。
func (r *ServerRuntime) PositionService() positiondomain.Service {
	return r.positionService
}

func buildBootstrapUsers(items []config.BootstrapUserConfig) []userdomain.BootstrapUser {
	result := make([]userdomain.BootstrapUser, 0, len(items))
	for _, item := range items {
		result = append(result, userdomain.BootstrapUser{
			Username:     item.Username,
			DisplayName:  item.DisplayName,
			PasswordHash: item.PasswordHash,
			Status:       item.Status,
			Role:         item.Role,
		})
	}

	return result
}

func buildSessionOptions(cfg *config.Config) (sessions.Options, error) {
	sameSiteMode, err := parseSameSiteMode(cfg.Auth.SessionSameSite)
	if err != nil {
		return sessions.Options{}, err
	}
	if sameSiteMode == http.SameSiteNoneMode && !cfg.Auth.SessionSecure {
		return sessions.Options{}, errors.New("auth session_secure must be true when session_same_site is none")
	}

	return sessions.Options{
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   cfg.Auth.SessionSecure,
		SameSite: sameSiteMode,
	}, nil
}

func parseSameSiteMode(value string) (http.SameSite, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	default:
		return http.SameSiteDefaultMode, fmt.Errorf("unsupported auth session_same_site %q", value)
	}
}
