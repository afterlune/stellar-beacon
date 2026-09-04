package task

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/row"
	"benetnasch/app/infra/shared"
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"log/slog"
	"strings"
	"time"
)

func StatisticsUserArea(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(time.Minute * 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		var users []row.TUserAuth
		engine := ormInit.GetEngine()
		if engine == nil {
			return fmt.Errorf("database engine is not initialized")
		}
		err := engine.Context(ctx).Prepare().Find(&users)
		if err != nil {
			slog.Error("load users for area statistics failed", "error_code", safeWorkerError(err))
			continue
		}
		var userAreas []port.UserAreaDTO
		hm := make(map[string]int64)
		for _, v := range users {
			if province, ok := provinceFromIPSource(v.IpSource); ok {
				hm[province]++
			}
		}
		for k, v := range hm {
			userAreas = append(userAreas, port.UserAreaDTO{
				Name:  k,
				Value: v,
			})
		}
		marshal, err := json.Marshal(userAreas)
		if err != nil {
			slog.Error("marshal user area statistics failed", "error_code", safeWorkerError(err))
			continue
		}
		if err := shared.SetCtx(ctx, shared.USER_AREA, marshal); err != nil {
			slog.Error("store user area statistics failed", "error_code", safeWorkerError(err))
			continue
		}
		slog.Info("user area statistics completed")
	}
}

func provinceFromIPSource(ipSource string) (string, bool) {
	parts := strings.Split(ipSource, "|")
	if len(parts) < 3 {
		return "", false
	}
	region := strings.TrimSpace(parts[2])
	if region == "" {
		return "", false
	}
	return strings.TrimSuffix(region, "省"), true
}
