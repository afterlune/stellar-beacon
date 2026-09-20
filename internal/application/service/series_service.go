package service

import (
	"strconv"
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

// SeriesService owns ordered article collections: public reads plus the admin
// CRUD that backs the 系列管理 page.
type SeriesService interface {
	ListPublicSeries(c *gin.Context) model.ResultVO
	GetPublicSeries(c *gin.Context) model.ResultVO
	ListAdminSeries(c *gin.Context) model.ResultVO
	SaveOrUpdateSeries(c *gin.Context) model.ResultVO
	DeleteSeries(c *gin.Context) model.ResultVO
	ListSeriesOptions(c *gin.Context) model.ResultVO
}

type MySeriesService struct {
	repo     port.SeriesRepository
	articles port.ArticleRepository
}

func NewSeriesService(deps SeriesServiceDeps) (*MySeriesService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MySeriesService{repo: deps.Repo, articles: deps.Articles}, nil
}

// SeriesDetailDTO is the public series page payload: the collection plus its
// published articles in the authored order.
type SeriesDetailDTO struct {
	Series   port.Series         `json:"series"`
	Articles []*port.ArticleCard `json:"articles"`
}

func (s *MySeriesService) ListPublicSeries(c *gin.Context) model.ResultVO {
	series, err := s.repo.ListPublic(c.Request.Context())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(series)
}

func (s *MySeriesService) GetPublicSeries(c *gin.Context) model.ResultVO {
	seriesID, err := strconv.Atoi(c.Param("seriesId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, err := s.repo.Get(c.Request.Context(), seriesID)
	if err != nil {
		return model.ResultFromError(err)
	}
	if series.Status != 1 || series.ModerationStatus == "hidden" {
		return model.ResultFailWithMessage("系列不存在")
	}
	articles, err := s.articles.ListArticleCardsBySeries(c.Request.Context(), seriesID)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(SeriesDetailDTO{
		Series:   port.Series{Id: series.Id, SeriesName: series.SeriesName, SeriesDesc: series.SeriesDesc, Cover: series.Cover, ArticleCount: len(articles)},
		Articles: articles,
	})
}

func (s *MySeriesService) ListAdminSeries(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, total, err := s.repo.ListAdmin(c.Request.Context(), current, size, c.Query("keywords"))
	if err != nil {
		return model.ResultFromError(err)
	}
	if len(series) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: []*port.Series{}, Count: int(total)})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: series, Count: int(total)})
}

func (s *MySeriesService) ListSeriesOptions(c *gin.Context) model.ResultVO {
	options, err := s.repo.ListOptions(c.Request.Context())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(options)
}

func (s *MySeriesService) SaveOrUpdateSeries(c *gin.Context) model.ResultVO {
	var vo model.SeriesVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	name := strings.TrimSpace(vo.SeriesName)
	if name == "" || len([]rune(name)) > 50 {
		return model.ResultFromError(apperrors.Invalid("series.save", "series name is required"))
	}
	if len([]rune(vo.SeriesDesc)) > 255 {
		return model.ResultFromError(apperrors.Invalid("series.save", "series description is too long"))
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFailWithMessage("用户未登录")
	}
	user, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFailWithMessage("用户信息无效")
	}
	ownerID := user.UserInfoId
	if vo.Id != 0 {
		if existing, existingErr := s.repo.Get(c.Request.Context(), vo.Id); existingErr == nil {
			ownerID = existing.UserId
		}
	}
	series, err := s.repo.SaveOrUpdate(c.Request.Context(), entity.TSeries{
		Id: vo.Id, UserId: ownerID, SeriesName: name, SeriesDesc: strings.TrimSpace(vo.SeriesDesc), Cover: strings.TrimSpace(vo.Cover), Status: 1,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(series)
}

func (s *MySeriesService) DeleteSeries(c *gin.Context) model.ResultVO {
	ids := make([]int, 0, 1)
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if len(ids) == 0 {
		return model.ResultFromError(apperrors.Invalid("series.delete", "series id is required"))
	}
	for _, id := range ids {
		if err := s.repo.Delete(c.Request.Context(), id); err != nil {
			if apperrors.IsKind(err, apperrors.KindNotFound) {
				continue
			}
			return model.ResultFromError(err)
		}
	}
	return model.ResultOk()
}

var _ SeriesService = (*MySeriesService)(nil)
