package task

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"github.com/goccy/go-json"
	"log/slog"
	"strings"
	"time"
)

func StatisticsUserArea() {
	ticker := time.NewTicker(time.Minute * 3)
	for range ticker.C {
		var users []entity.TUserAuth
		err := ormInit.GetEngine().Prepare().Find(&users)
		if err != nil {
			slog.Error("load users for area statistics failed", "error", err)
			continue
		}
		var userAreas []model.UserAreaDTO
		hm := make(map[string]int64)
		for _, v := range users {
			s := strings.Split(v.IpSource, "|")
			if len(s) > 2 {
				region := strings.Split(v.IpSource, "|")[2]
				province := region
				if strings.HasSuffix(region, "省") {
					province = strings.Split(region, "省")[0]
				}
				hm[province] = hm[province] + 1
			}
		}
		for k, v := range hm {
			userAreas = append(userAreas, model.UserAreaDTO{
				Name:  k,
				Value: v,
			})
		}
		marshal, err := json.Marshal(userAreas)
		if err != nil {
			slog.Error("marshal user area statistics failed", "error", err)
			continue
		}
		shared.Set(shared.USER_AREA, marshal)
		slog.Info("user area statistics completed")
	}
}
