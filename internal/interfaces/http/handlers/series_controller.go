package api

import (
	"net/http"
	"strconv"

	"github.com/afterlune/stellar-beacon/internal/application/service"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListPublicSeries
// @Summary         文章系列模块
// @Description    获取公开系列列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/public/series [GET]
func ListPublicSeries(c *gin.Context) {
	series, err := seriesService.ListPublicSeries(c.Request.Context())
	writeSeriesResult(c, series, err)
}

// GetPublicSeries
// @Summary         文章系列模块
// @Description    获取系列详情与按序文章
// @Success        200 {object} model.ResultVO
// @Router         /v1/public/series/{seriesId} [GET]
func GetPublicSeries(c *gin.Context) {
	seriesID, err := strconv.Atoi(c.Param("seriesId"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	detail, err := seriesService.GetPublicSeries(c.Request.Context(), seriesID)
	if apperrors.Op(err) == "series.public.visibility" {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("系列不存在"))
		return
	}
	writeSeriesResult(c, detail, err)
}

// ListAdminSeries
// @Summary         文章系列模块
// @Description    后台系列列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/series [GET]
func ListAdminSeries(c *gin.Context) {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	status, err := strconv.Atoi(c.DefaultQuery("status", "0"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	series, total, err := seriesService.ListAdminSeries(c.Request.Context(), port.SeriesFilter{
		Current: current, Size: size, Keywords: c.Query("keywords"), Status: status, ModerationStatus: c.Query("moderationStatus"),
	})
	if err != nil {
		writeSeriesResult(c, nil, err)
		return
	}
	if len(series) == 0 {
		series = []*port.Series{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: series, Count: int(total)}))
}

// ListSeriesOptions
// @Summary         文章系列模块
// @Description    系列下拉选项
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/series/options [GET]
func ListSeriesOptions(c *gin.Context) {
	options, err := seriesService.ListSeriesOptions(c.Request.Context())
	writeSeriesResult(c, options, err)
}

// SaveOrUpdateSeries
// @Summary         文章系列模块
// @Description    新增或修改系列
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/series [POST]
func SaveOrUpdateSeries(c *gin.Context) {
	var request model.SeriesVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("用户未登录"))
		return
	}
	series, err := seriesService.SaveOrUpdateSeries(c.Request.Context(), userID, service.SeriesSaveInput{
		ID: request.Id, Name: request.SeriesName, Description: request.SeriesDesc, Cover: request.Cover,
	})
	writeSeriesResult(c, series, err)
}

// DeleteSeries
// @Summary         文章系列模块
// @Description    删除系列并解绑文章
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/series [DELETE]
func DeleteSeries(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	writeSeriesResult(c, nil, seriesService.DeleteSeries(c.Request.Context(), userID, ids))
}

func writeSeriesResult(c *gin.Context, data any, err error) {
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if data == nil {
		c.JSON(http.StatusOK, model.ResultOk())
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(data))
}
