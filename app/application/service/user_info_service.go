package service

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/shared"
	"container/list"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type UserInfoService interface {
	UpdateUserInfo(c *gin.Context) model.ResultVO
	UpdateUserAvatar(c *gin.Context) model.ResultVO
	SaveUserEmail(c *gin.Context) model.ResultVO
	UpdateUserSubscribe(c *gin.Context) model.ResultVO
	UpdateUserRole(c *gin.Context) model.ResultVO
	UpdateUserDisable(c *gin.Context) model.ResultVO
	ListOnlineUsers(c *gin.Context) model.ResultVO
	RemoveOnlineUser(c *gin.Context) model.ResultVO
	GetUserInfoById(c *gin.Context) model.ResultVO
}

type MyUserInfoService struct{ repo port.UserInfoRepository }

func NewUserInfoService(repo port.UserInfoRepository) *MyUserInfoService {
	return &MyUserInfoService{repo: repo}
}

func (u *MyUserInfoService) userInfoRepository() port.UserInfoRepository {
	if u.repo != nil {
		return u.repo
	}
	return userInfoRepo
}

func (u *MyUserInfoService) UpdateUserInfo(c *gin.Context) model.ResultVO {
	var vo model.UserInfoVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	if err := u.userInfoRepository().UpdateProfile(c.Request.Context(), dto.UserInfoId, vo.Nickname, vo.Intro, vo.Website); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserAvatar(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	fileURI := oss.Upload(file, "avatar/")
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	avatar := shared.FILEURL + fileURI
	if err := u.userInfoRepository().UpdateAvatar(c.Request.Context(), dto.UserInfoId, avatar); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(avatar)
}

func (u *MyUserInfoService) SaveUserEmail(c *gin.Context) model.ResultVO {
	var vo model.EmailVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	vo.Email = strings.ToLower(strings.TrimSpace(vo.Email))
	code, err := shared.GetCtx(c.Request.Context(), shared.USER_CODE_KEY+vo.Email)
	if err != nil {
		return model.ResultFail()
	}
	if code == "" || code != vo.Code {
		return model.ResultFailWithMessage("验证码错误")
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	if err := u.userInfoRepository().UpdateEmail(c.Request.Context(), dto.UserInfoId, vo.Email); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserSubscribe(c *gin.Context) model.ResultVO {
	var vo model.SubscribeVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	info, err := u.userInfoRepository().GetByID(c.Request.Context(), vo.UserId)
	if err != nil {
		return model.ResultFromError(err)
	}
	if info.Email == "" {
		return model.ResultFailWithMessage("邮箱未绑定！")
	}
	if err := u.userInfoRepository().UpdateSubscribe(c.Request.Context(), vo.UserId, vo.IsSubscribe); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserRole(c *gin.Context) model.ResultVO {
	var vo model.UserRoleVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := u.userInfoRepository().UpdateRole(c.Request.Context(), vo.UserInfoId, vo.NickName, vo.RoleIds); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserDisable(c *gin.Context) model.ResultVO {
	var vo model.UserDetailsDTO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := u.userInfoRepository().UpdateDisable(c.Request.Context(), vo.Id, vo.IsDisable); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) ListOnlineUsers(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	userMaps, err := shared.HGetAllCtx(c.Request.Context(), shared.LOGIN_USER)
	if err != nil {
		return model.ResultFail()
	}
	users := make([]model.UserDetailsDTO, 0, len(userMaps))
	for _, value := range userMaps {
		var dto model.UserDetailsDTO
		if err := shared.Unmarsh(value, &dto); err != nil {
			slog.WarnContext(c.Request.Context(), "skip malformed online user cache", "error", err)
			continue
		}
		users = append(users, dto)
	}
	var online []model.UserOnlineDTO
	shared.StructCopy(users, &online)
	filtered := online[:0]
	for _, user := range online {
		if vo.Keywords == "" || strings.Contains(user.Nickname, vo.Keywords) {
			filtered = append(filtered, user)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].LastLoginTime.After(filtered[j].LastLoginTime) })
	if vo.Current < 1 {
		vo.Current = 1
	}
	if vo.Size < 1 {
		vo.Size = 10
	}
	total := len(filtered)
	from := (vo.Current - 1) * vo.Size
	if from >= total {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: total})
	}
	to := from + vo.Size
	if to > total {
		to = total
	}
	if total == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: filtered[from:to], Count: total})
}

func (u *MyUserInfoService) RemoveOnlineUser(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("userInfoId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	auth, err := u.userInfoRepository().FindAuthByUserInfoID(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	if err := shared.HDelCtx(c.Request.Context(), shared.LOGIN_USER, strconv.Itoa(auth.Id)); err != nil {
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) GetUserInfoById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("userInfoId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	info, err := u.userInfoRepository().GetByID(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dto model.UserInfoDTO
	shared.StructCopy(info, &dto)
	return model.ResultOkWithData(dto)
}
