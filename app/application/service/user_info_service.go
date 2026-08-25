package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"sort"
	"strconv"
	"strings"
	"xorm.io/xorm"
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

type MyUserInfoService struct{}

func (u *MyUserInfoService) UpdateUserInfo(c *gin.Context) model.ResultVO {
	var userInfoVO model.UserInfoVO
	err := c.ShouldBind(&userInfoVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	userinfo := entity.TUserInfo{
		Id:       dto.UserInfoId,
		Nickname: userInfoVO.Nickname,
		Intro:    userInfoVO.Intro,
		Website:  userInfoVO.Website,
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Exec("update t_user_info set nickname = ?, intro = ?, website = ? where id = ?", userinfo.Nickname, userinfo.Intro, userinfo.Website, userinfo.Id)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserAvatar(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	fileUri := oss.Upload(file, "avatar/")
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	userinfo := entity.TUserInfo{
		Id:     dto.UserInfoId,
		Avatar: shared.FILEURL + fileUri,
	}

	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Exec("update t_user_info set avatar = ? where id = ?", userinfo.Avatar, userinfo.Id)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage(err.Error())
	}
	return model.ResultOkWithData(shared.FILEURL + fileUri)
}

func (u *MyUserInfoService) SaveUserEmail(c *gin.Context) model.ResultVO {
	var vo model.EmailVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	vo.Email = strings.ToLower(strings.TrimSpace(vo.Email))
	code, err := shared.GetCtx(c.Request.Context(), shared.USER_CODE_KEY+vo.Email)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if code == "" || code != vo.Code {
		return model.ResultFailWithMessage("验证码错误")
	}
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	userInfo := entity.TUserInfo{
		Id:    dto.UserInfoId,
		Email: vo.Email,
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.ID(userInfo.Id).Update(&userInfo)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserSubscribe(c *gin.Context) model.ResultVO {
	var subVO model.SubscribeVO
	err := c.ShouldBind(&subVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var userinfo entity.TUserInfo
	_, err = engine.Context(c.Request.Context()).ID(subVO.UserId).Get(&userinfo)
	if err != nil {
		zlog.Error(err.Error())
	}
	if userinfo.Email == "" {
		return model.ResultFailWithMessage("邮箱未绑定！")
	}
	userinfo.Id = subVO.UserId
	userinfo.IsSubscribe = subVO.IsSubscribe

	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Exec("update t_user_info set is_subscribe = ? where id = ?", userinfo.IsSubscribe, userinfo.Id)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage(err.Error())
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserRole(c *gin.Context) model.ResultVO {
	var vo model.UserRoleVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	userInfo := entity.TUserInfo{
		Id:       vo.UserInfoId,
		Nickname: vo.NickName,
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		if _, err := session.ID(userInfo.Id).Update(&userInfo); err != nil {
			return err
		}
		if _, err := session.Where("user_id = ?", vo.UserInfoId).Delete(&entity.TUserRole{}); err != nil {
			return err
		}
		userRoles := make([]entity.TUserRole, 0, len(vo.RoleIds))
		for _, v := range vo.RoleIds {
			userRoles = append(userRoles, entity.TUserRole{RoleId: v, UserId: vo.UserInfoId})
		}
		if len(userRoles) == 0 {
			return nil
		}
		_, err := session.Insert(&userRoles)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserDisable(c *gin.Context) model.ResultVO {
	var vo model.UserDetailsDTO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	userInfo := entity.TUserInfo{
		Id:        vo.Id,
		IsDisable: vo.IsDisable,
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.ID(userInfo.Id).MustCols("is_disable").Update(&userInfo)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) ListOnlineUsers(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	userMaps, err := shared.HGetAllCtx(c.Request.Context(), shared.LOGIN_USER)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var userDetailsDTOs []model.UserDetailsDTO
	for _, v := range userMaps {
		var dto model.UserDetailsDTO
		shared.Unmarsh(v, &dto)
		userDetailsDTOs = append(userDetailsDTOs, dto)
	}
	var userOnlineDTOs []model.UserOnlineDTO
	shared.StructCopy(userDetailsDTOs, &userOnlineDTOs)
	var onlineUsers []model.UserOnlineDTO
	for _, v := range userOnlineDTOs {
		if vo.Keywords == "" || strings.Contains(v.Nickname, vo.Keywords) {
			onlineUsers = append(onlineUsers, v)
		}
	}
	sort.Slice(onlineUsers, func(i, j int) bool {
		return onlineUsers[i].LastLoginTime.After(onlineUsers[j].LastLoginTime)
	})
	if vo.Current < 1 {
		vo.Current = 1
	}
	if vo.Size < 1 {
		vo.Size = 10
	}
	fromIndex := (vo.Current - 1) * vo.Size
	if fromIndex >= len(onlineUsers) {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: len(onlineUsers)})
	}
	toIndex := 0
	n := len(onlineUsers)
	if (n - fromIndex) > vo.Size {
		toIndex = fromIndex + vo.Size
	} else {
		toIndex = n
	}
	onlineUsers = onlineUsers[fromIndex:toIndex]
	if n == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: onlineUsers, Count: n})
}

func (u *MyUserInfoService) RemoveOnlineUser(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("userInfoId"))
	var userAuth entity.TUserAuth
	_, err := ormInit.GetEngine().Prepare().Where("user_info_id = ?", id).Get(&userAuth)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if err := shared.HDelCtx(c.Request.Context(), shared.LOGIN_USER, strconv.Itoa(userAuth.Id)); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) GetUserInfoById(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("userInfoId"))
	var userInfo entity.TUserInfo
	_, err := ormInit.GetEngine().Prepare().ID(id).Get(&userInfo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var userInfoDTO model.UserInfoDTO
	shared.StructCopy(userInfo, &userInfoDTO)
	return model.ResultOkWithData(userInfoDTO)
}
