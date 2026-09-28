package service

import (
	"strconv"
	"strings"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func currentUser(c *gin.Context) (model.UserDetailsDTO, bool) {
	value, ok := c.Get("userInfo")
	if !ok {
		return model.UserDetailsDTO{}, false
	}
	user, ok := value.(model.UserDetailsDTO)
	if !ok || user.UserInfoId <= 0 {
		return model.UserDetailsDTO{}, false
	}
	return user, true
}

func pageParams(c *gin.Context) (int, int, error) {
	current, err := strconv.Atoi(c.DefaultQuery("current", "1"))
	if err != nil || current < 1 {
		return 0, 0, errInvalidPage
	}
	size, err := strconv.Atoi(c.DefaultQuery("size", "12"))
	if err != nil || size < 1 {
		return 0, 0, errInvalidPage
	}
	if size > 100 {
		size = 100
	}
	return current, size, nil
}

func studioFilter(c *gin.Context) (port.StudioFilter, error) {
	current, size, err := pageParams(c)
	if err != nil {
		return port.StudioFilter{}, err
	}
	status := 0
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status, err = strconv.Atoi(raw)
		if err != nil {
			return port.StudioFilter{}, errInvalidPage
		}
	}
	seriesID := 0
	if raw := strings.TrimSpace(c.Query("seriesId")); raw != "" {
		seriesID, err = strconv.Atoi(raw)
		if err != nil || seriesID <= 0 {
			return port.StudioFilter{}, errInvalidPage
		}
	}
	return port.StudioFilter{Current: current, Size: size, Status: status, SeriesID: seriesID, Keywords: c.Query("keywords")}, nil
}

func pathID(c *gin.Context, name string) (int, error) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		return 0, errInvalidPage
	}
	return id, nil
}

func visibilityStatus(value string, article bool) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "public":
		return 1, true
	case "private":
		return 2, true
	case "draft":
		return 3, true
	case "scheduled":
		return 4, article
	default:
		return 0, false
	}
}

var errInvalidPage = &strconv.NumError{Func: "page", Num: "", Err: strconv.ErrSyntax}
