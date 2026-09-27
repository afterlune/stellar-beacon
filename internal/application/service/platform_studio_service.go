package service

import (
	"net/url"
	"strings"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func (s *MyPlatformService) Dashboard(c *gin.Context) model.ResultVO {
	dto, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	dashboard, err := s.platformRepo().StudioDashboard(c.Request.Context(), dto.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(dashboard)
}

func (s *MyPlatformService) SyncActivation(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioActivationVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	activation, err := s.platformRepo().SyncStudioActivation(c.Request.Context(), user.UserInfoId, port.StudioActivationUpdate{
		Started: vo.Started, Collapsed: vo.Collapsed, IdentityComplete: vo.IdentityComplete,
		ContentComplete: vo.ContentComplete, ProfileVisited: vo.ProfileVisited, Completed: vo.Completed,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(activation)
}

func (s *MyPlatformService) GetProfile(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	profile, err := s.platformRepo().GetStudioProfile(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(studioProfileDTO(profile))
}

func (s *MyPlatformService) UpdateProfile(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioProfileVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	profile, message := normalizeStudioProfile(vo)
	if message != "" {
		return model.ResultFailWithMessage(message)
	}
	if err := s.platformRepo().UpdateAuthorProfile(c.Request.Context(), user.UserInfoId, port.StudioProfile{
		Handle: profile.Handle, Nickname: profile.Nickname, Intro: profile.Intro,
		Website: profile.Website, About: profile.About, Links: profile.Links,
	}); err != nil {
		if apperrors.KindOf(err) == apperrors.KindConflict {
			return model.ResultFailWithMessage("该 Handle 已被占用")
		}
		return model.ResultFromError(err)
	}
	saved, err := s.platformRepo().GetStudioProfile(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(studioProfileDTO(saved))
}

func normalizeStudioProfile(vo model.StudioProfileVO) (model.StudioProfileVO, string) {
	vo.Handle = strings.ToLower(strings.TrimSpace(vo.Handle))
	vo.Nickname = strings.TrimSpace(vo.Nickname)
	vo.Intro = strings.TrimSpace(vo.Intro)
	vo.Website = strings.TrimSpace(vo.Website)
	vo.About = strings.TrimSpace(vo.About)
	if !validStudioHandle(vo.Handle) {
		return vo, "Handle 需为 3-40 位小写字母、数字或连字符，且必须以字母或数字开头"
	}
	if vo.Nickname == "" {
		return vo, "昵称不能为空"
	}
	if len([]rune(vo.Nickname)) > 30 {
		return vo, "昵称不能超过 30 个字"
	}
	if len([]rune(vo.Intro)) > 255 {
		return vo, "个人简介不能超过 255 个字"
	}
	if len([]rune(vo.Website)) > 255 {
		return vo, "个人网站不能超过 255 个字"
	}
	if len([]rune(vo.About)) > 20000 {
		return vo, "主页介绍不能超过 20000 个字"
	}
	if vo.Website != "" && !validStudioWebsite(vo.Website) {
		return vo, "个人网站必须是有效的 HTTP(S) 地址"
	}
	links := make([]port.ProfileLink, 0, len(vo.Links))
	for _, link := range vo.Links {
		link.Label = strings.TrimSpace(link.Label)
		link.URL = strings.TrimSpace(link.URL)
		link.Description = strings.TrimSpace(link.Description)
		if link.Label == "" && link.URL == "" && link.Description == "" {
			continue
		}
		if link.Label == "" || len([]rune(link.Label)) > 40 {
			return vo, "每条外链都需要填写不超过 40 个字的名称"
		}
		if !validStudioWebsite(link.URL) {
			return vo, "个人外链必须是有效的 HTTP(S) 地址"
		}
		if len([]rune(link.Description)) > 160 {
			return vo, "外链说明不能超过 160 个字"
		}
		links = append(links, link)
	}
	if len(links) > 100 {
		return vo, "个人外链不能超过 100 条"
	}
	vo.Links = links
	return vo, ""
}

func validStudioHandle(value string) bool {
	if len(value) < 3 || len(value) > 40 {
		return false
	}
	for index, r := range value {
		if index == 0 && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func validStudioWebsite(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func studioProfileDTO(profile port.StudioProfile) model.StudioProfileDTO {
	return model.StudioProfileDTO{
		Handle: profile.Handle, Nickname: profile.Nickname, Avatar: profile.Avatar,
		Intro: profile.Intro, Website: profile.Website, About: profile.About,
		Links: profile.Links,
	}
}
