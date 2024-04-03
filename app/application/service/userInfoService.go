package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"fmt"
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
	}
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	userinfo := entity.TUserInfo{
		Id:       dto.UserInfoId,
		Nickname: userInfoVO.Nickname,
		Intro:    userInfoVO.Intro,
		Website:  userInfoVO.Website,
	}
	session := ormInit.GetEngine().NewSession()
	err = session.Begin()
	if err != nil {
		zlog.Error(err.Error())
	}
	defer func(session *xorm.Session) {
		err := session.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(session)
	sql := fmt.Sprintf("update t_user_info set nickname = '%s', intro = '%s', website = '%s' where id = %d", userinfo.Nickname, userinfo.Intro, userinfo.Website, userinfo.Id)
	_, err = session.Prepare().Exec(sql)
	if err != nil {
		err = session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFail()
	}
	err = session.Commit()
	if err != nil {
		err = session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserAvatar(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())

	}
	fileUri := oss.Upload(file, "avatar/")
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	userinfo := entity.TUserInfo{
		Id:     dto.UserInfoId,
		Avatar: shared.FILEURL + fileUri,
	}

	session := ormInit.GetEngine().NewSession()
	err = session.Begin()
	if err != nil {
		zlog.Error(err.Error())
	}
	defer func(session *xorm.Session) {
		err := session.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(session)
	sql := fmt.Sprintf("update t_user_info set avatar = '%s' where id = %d", userinfo.Avatar, userinfo.Id)
	_, err = session.Prepare().Exec(sql)
	if err != nil {
		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFailWithMessage(err.Error())
	}
	err = session.Commit()
	if err != nil {
		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
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
	if shared.Get(shared.USER_CODE_KEY+vo.Email) == "" || shared.Get(shared.USER_CODE_KEY+vo.Email) != vo.Code {
		return model.ResultFailWithMessage("验证码错误")
	}
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	userInfo := entity.TUserInfo{
		Id:    dto.UserInfoId,
		Email: vo.Email,
	}
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	_, err = session.Prepare().ID(userInfo.Id).Update(&userInfo)
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func (u *MyUserInfoService) UpdateUserSubscribe(c *gin.Context) model.ResultVO {
	var subVO model.SubscribeVO
	err := c.ShouldBind(&subVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine()
	var userinfo entity.TUserInfo
	_, err = engine.ID(subVO.UserId).Get(&userinfo)
	if err != nil {
		zlog.Error(err.Error())
	}
	if userinfo.Email == "" {
		return model.ResultFailWithMessage("邮箱未绑定！")
	}
	userinfo.Id = subVO.UserId
	userinfo.IsSubscribe = subVO.IsSubscribe

	session := engine.NewSession()
	err = session.Begin()
	if err != nil {
		zlog.Error(err.Error())
	}
	defer func(session *xorm.Session) {
		err := session.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(session)
	sql := fmt.Sprintf("update t_user_info set is_subscribe = %d where id = %d", userinfo.IsSubscribe, userinfo.Id)
	_, err = session.Prepare().Exec(sql)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if err != nil {
		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFailWithMessage(err.Error())
	}
	err = session.Commit()
	if err != nil {
		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
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
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	_, err = session.Prepare().ID(userInfo.Id).Update(&userInfo)
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	_, err = session.Where(fmt.Sprintf("user_id = %d", vo.UserInfoId)).Delete(&entity.TUserRole{})
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	var userRoles []entity.TUserRole
	for _, v := range vo.RoleIds {
		userRoles = append(userRoles, entity.TUserRole{
			RoleId: v,
			UserId: vo.UserInfoId,
		})
	}
	_, err = session.Prepare().Insert(&userRoles)
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	session.Commit()
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
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	_, err = session.Prepare().ID(userInfo.Id).MustCols("is_disable").Update(&userInfo)
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func (u *MyUserInfoService) ListOnlineUsers(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	userMaps := shared.HGetAll(shared.LOGIN_USER)
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
		if vo.Keywords != "" || strings.Contains(v.Nickname, vo.Keywords) {
			onlineUsers = append(onlineUsers, v)
		}
	}
	sort.Slice(onlineUsers, func(i, j int) bool {
		return onlineUsers[i].LastLoginTime.After(onlineUsers[j].LastLoginTime)
	})
	fromIndex := (vo.Current - 1) * vo.Size
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
	_, err := ormInit.GetEngine().Prepare().Where(fmt.Sprintf("user_info_id = %d", id)).Get(&userAuth)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	shared.HDel(shared.LOGIN_USER, strconv.Itoa(userAuth.Id))
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
