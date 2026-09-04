package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

type UserInfoService interface {
	UpdateUserInfo(c port.Request) port.ResultVO
	UpdateUserAvatar(c port.Request) port.ResultVO
	SaveUserEmail(c port.Request) port.ResultVO
	UpdateUserSubscribe(c port.Request) port.ResultVO
	UpdateUserRole(c port.Request) port.ResultVO
	UpdateUserDisable(c port.Request) port.ResultVO
	ListOnlineUsers(c port.Request) port.ResultVO
	RemoveOnlineUser(c port.Request) port.ResultVO
	GetUserInfoById(c port.Request) port.ResultVO
}

type MyUserInfoService struct {
	repo    port.UserInfoRepository
	cache   port.Cache
	storage port.ObjectStorage
}

func NewUserInfoService(deps UserInfoServiceDeps) (*MyUserInfoService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyUserInfoService{
		repo:    deps.Repo,
		cache:   deps.Cache,
		storage: deps.Storage,
	}, nil
}

func (u *MyUserInfoService) userInfoRepository() port.UserInfoRepository {
	return u.repo
}

func (u *MyUserInfoService) UpdateUserInfo(c port.Request) port.ResultVO {
	var vo port.UserInfoVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return port.ResultFailWithStatus(port.NO_LOGIN)
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok {
		return port.ResultFailWithStatus(port.NO_LOGIN)
	}
	if err := u.userInfoRepository().UpdateProfile(c.Context(), dto.UserInfoId, vo.Nickname, vo.Intro, vo.Website); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (u *MyUserInfoService) UpdateUserAvatar(c port.Request) port.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	ref, err := uploadMultipart(c.Context(), u.storage, file, "avatar/")
	if err != nil {
		return port.ResultFromError(err)
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return port.ResultFailWithStatus(port.NO_LOGIN)
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok {
		return port.ResultFailWithStatus(port.NO_LOGIN)
	}
	avatar := ref.URL
	if err := u.userInfoRepository().UpdateAvatar(c.Context(), dto.UserInfoId, avatar); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(avatar)
}

func (u *MyUserInfoService) SaveUserEmail(c port.Request) port.ResultVO {
	var vo port.EmailVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	vo.Email = strings.ToLower(strings.TrimSpace(vo.Email))
	if u.cache == nil {
		return port.ResultFail()
	}
	code, err := u.cache.Get(c.Context(), support.UserCodeKey+vo.Email)
	if err != nil {
		if errors.Is(err, port.ErrCacheMiss) {
			return port.ResultFailWithMessage("验证码错误")
		}
		return port.ResultFail()
	}
	if code == "" || code != vo.Code {
		return port.ResultFailWithMessage("验证码错误")
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return port.ResultFailWithStatus(port.NO_LOGIN)
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok {
		return port.ResultFailWithStatus(port.NO_LOGIN)
	}
	if err := u.userInfoRepository().UpdateEmail(c.Context(), dto.UserInfoId, vo.Email); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (u *MyUserInfoService) UpdateUserSubscribe(c port.Request) port.ResultVO {
	var vo port.SubscribeVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	info, err := u.userInfoRepository().GetByID(c.Context(), vo.UserId)
	if err != nil {
		return port.ResultFromError(err)
	}
	if info.Email == "" {
		return port.ResultFailWithMessage("邮箱未绑定！")
	}
	if err := u.userInfoRepository().UpdateSubscribe(c.Context(), vo.UserId, vo.IsSubscribe); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (u *MyUserInfoService) UpdateUserRole(c port.Request) port.ResultVO {
	var vo port.UserRoleVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := u.userInfoRepository().UpdateRole(c.Context(), vo.UserInfoId, vo.NickName, vo.RoleIds); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (u *MyUserInfoService) UpdateUserDisable(c port.Request) port.ResultVO {
	var vo port.UserDetailsDTO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := u.userInfoRepository().UpdateDisable(c.Context(), vo.Id, vo.IsDisable); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (u *MyUserInfoService) ListOnlineUsers(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if u.cache == nil {
		return port.ResultFail()
	}
	userMaps, err := u.cache.HGetAll(c.Context(), support.LoginUser)
	if err != nil {
		return port.ResultFail()
	}
	users := make([]port.UserDetailsDTO, 0, len(userMaps))
	for _, value := range userMaps {
		var dto port.UserDetailsDTO
		if err := support.Unmarsh(value, &dto); err != nil {
			slog.WarnContext(c.Context(), "skip malformed online user cache", "error_code", apperrors.SafeCode(err))
			continue
		}
		users = append(users, dto)
	}
	var online []port.UserOnlineDTO
	support.StructCopy(users, &online)
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
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: total})
	}
	to := from + vo.Size
	if to > total {
		to = total
	}
	if total == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: filtered[from:to], Count: total})
}

func (u *MyUserInfoService) RemoveOnlineUser(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("userInfoId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	auth, err := u.userInfoRepository().FindAuthByUserInfoID(c.Context(), id)
	if err != nil {
		return port.ResultFromError(err)
	}
	if u.cache == nil {
		return port.ResultFail()
	}
	if err := u.cache.HDel(c.Context(), support.LoginUser, strconv.Itoa(auth.Id)); err != nil {
		return port.ResultFail()
	}
	return port.ResultOk()
}

func (u *MyUserInfoService) GetUserInfoById(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("userInfoId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	info, err := u.userInfoRepository().GetByID(c.Context(), id)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dto port.UserInfoDTO
	support.StructCopy(info, &dto)
	return port.ResultOkWithData(dto)
}
