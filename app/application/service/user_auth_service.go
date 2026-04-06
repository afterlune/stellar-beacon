package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm"
)

type UserAuthService interface {
	SendCode(c *gin.Context) model.ResultVO
	ListUserAreas(c *gin.Context) model.ResultVO
	ListUsers(c *gin.Context) model.ResultVO
	Register(c *gin.Context) model.ResultVO
	UpdatePassword(c *gin.Context) model.ResultVO
	UpdateAdminPassword(c *gin.Context) model.ResultVO
	Logout(id int) model.ResultVO
	QQLogin(c *gin.Context) model.ResultVO
	CheckUser(vo model.UserVO) bool
	CheckUserAuth(vo model.UserVO) *model.UserDetailsDTO
	UpdateUserIp(user entity.TUserAuth)
}

type MyUserAuthService struct{}

func (u *MyUserAuthService) SendCode(c *gin.Context) model.ResultVO {
	Username := c.Query("username")
	if !shared.CheckEmail(Username) {
		return model.ResultFailWithMessage("请输入正确邮箱")
	}
	code := shared.RandomCode()
	codem := make(map[string]interface{})
	codem["content"] = "您的验证码为 " + code + " 有效期15分钟，请不要告诉他人哦！"
	emailDTO := model.EmailDTO{
		Email:      Username,
		CommentMap: codem,
		Template:   "resource/template/common.html",
		Subject:    shared.CAPTCHA,
	}
	shared.SendHtmlEmail(emailDTO)
	shared.SetWithTime(shared.USER_CODE_KEY+Username, code, shared.CODE_EXPIRE_TIME)
	return model.ResultOk()
}

func (u *MyUserAuthService) ListUserAreas(c *gin.Context) model.ResultVO {
	typeId := c.Query("type")
	var userAreaDTOs []model.UserAreaDTO
	switch typeId {
	case "1":
		userArea := shared.Get(shared.USER_AREA).(string)
		if userArea != "" {
			err := json.Unmarshal([]byte(userArea), &userAreaDTOs)
			if err != nil {
				zlog.Error(err.Error())
			}
		}
		return model.ResultOkWithData(userAreaDTOs)
	case "2":
		visitorArea := shared.HGetAll(shared.VISITOR_AREA)
		if visitorArea != nil {
			for k, v := range visitorArea {
				visitCount, err := strconv.Atoi(v)
				region := strings.Split(k, "|")[2]
				province := region
				if strings.HasSuffix(region, "省") {
					province = strings.Split(region, "省")[0]
				}
				if err != nil {
					zlog.Error(err.Error())
				}
				userAreaDTO := model.UserAreaDTO{
					Name:  province,
					Value: int64(visitCount),
				}
				userAreaDTOs = append(userAreaDTOs, userAreaDTO)
			}
		}
	default:
		break
	}
	return model.ResultOkWithData(userAreaDTOs)
}

func (u *MyUserAuthService) ListUsers(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	count := userAuthRepo.CountUser(&vo)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	data := userAuthRepo.ListUsers(vo.Current, vo.Size, &vo)
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (u *MyUserAuthService) Register(c *gin.Context) model.ResultVO {
	var userVo model.UserVO
	err := c.ShouldBind(&userVo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	if userVo.Code != shared.Get(shared.USER_CODE_KEY+userVo.Username) {
		return model.ResultFailWithMessage("验证码有误！")
	}

	if !shared.CheckEmail(userVo.Username) {
		return model.ResultFailWithMessage("邮箱格式不对！")
	}
	if u.CheckUser(userVo) {
		return model.ResultFailWithMessage("邮箱已被注册！")
	}
	userInfo := entity.TUserInfo{
		Email:    userVo.Username,
		Nickname: shared.DEFAULT_NICKNAME,
		Avatar:   benetnaschService.GetWebsiteConfig().Data.(model.WebsiteConfigDTO).UserAvatar,
	}

	engine := ormInit.GetEngine()
	session := engine.NewSession()
	defer func(session *xorm.Session) {
		err := session.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(session)

	err = session.Begin()
	if err != nil {
		zlog.Error(err.Error())
	}

	_, err = session.Prepare().Insert(&userInfo)
	if err != nil {
		if err != nil {
			zlog.Error(err.Error())
		}
		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}
	userRole := entity.TUserRole{
		UserId: userInfo.Id,
		RoleId: 2,
	}
	_, err = session.Prepare().Insert(&userRole)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}
	password, err := bcrypt.GenerateFromPassword([]byte(userVo.Password), bcrypt.DefaultCost)
	if err != nil {
		if err != nil {
			zlog.Error(err.Error())
		}
		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}
	userAuth := entity.TUserAuth{
		UserInfoId: userInfo.Id,
		Username:   userVo.Username,
		Password:   string(password),
		LoginType:  1,
	}
	_, err = session.Prepare().Insert(&userAuth)
	if err != nil {
		if err != nil {
			zlog.Error(err.Error())
		}

		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}

	err = session.Commit()
	if err != nil {
		if err != nil {
			zlog.Error(err.Error())
		}

		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}
	return model.ResultOk()
}

func (u *MyUserAuthService) UpdatePassword(c *gin.Context) model.ResultVO {
	var userVO model.UserVO
	err := c.ShouldBind(&userVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	if userVO.Code != shared.Get(shared.USER_CODE_KEY+userVO.Username) {
		return model.ResultFailWithMessage("验证码有误！")
	}
	if !shared.CheckEmail(userVO.Username) {
		return model.ResultFailWithMessage("邮箱格式不对！")
	}
	if !u.CheckUser(userVO) {
		return model.ResultFailWithMessage("邮箱未注册！")
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userVO.Password), bcrypt.DefaultCost)
	if err != nil {
		zlog.Error(err.Error())
	}
	userAuth := entity.TUserAuth{
		Password: string(password),
		Username: userVO.Username,
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

	_, err = session.Prepare().Where("username = '" + userAuth.Username + "'").Update(&userAuth)
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

func (u *MyUserAuthService) UpdateAdminPassword(c *gin.Context) model.ResultVO {
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	var passwordVO model.PasswordVO
	err := c.ShouldBind(&passwordVO)
	if err != nil && passwordVO.NewPassword == "" || passwordVO.OldPassword == "" {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var user entity.TUserAuth
	_, err = engine.ID(dto.Id).Get(&user)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(passwordVO.OldPassword))
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	password, err := bcrypt.GenerateFromPassword([]byte(passwordVO.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if user.Username != "" && err == nil {
		userAuth := entity.TUserAuth{
			Id:       dto.Id,
			Password: string(password),
		}
		session := engine.NewSession()
		err := session.Begin()
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		defer func(session *xorm.Session) {
			err := session.Close()
			if err != nil {
				zlog.Error(err.Error())
			}
		}(session)
		_, err = session.Prepare().ID(userAuth.Id).Update(&userAuth)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		err = session.Commit()
		if err != nil {
			zlog.Unwrap(session.Rollback())
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		return model.ResultOk()
	}
	return model.ResultFailWithMessage("旧密码不正确")
}

func (u *MyUserAuthService) Logout(id int) model.ResultVO {
	shared.HDel(shared.LOGIN_USER, strconv.Itoa(id))
	return model.ResultOkWithData(model.UserLogoutStatusDTO{
		Message: "注销成功",
	})
}

func (u *MyUserAuthService) QQLogin(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (u *MyUserAuthService) CheckUser(vo model.UserVO) bool {
	var userAuth entity.TUserAuth
	b, err := ormInit.GetEngine().Where("Username = ?", vo.Username).Get(&userAuth)
	if err != nil {
		zlog.Error(err.Error())
		return false
	}
	return b
}

func (u *MyUserAuthService) CheckUserAuth(vo model.UserVO) *model.UserDetailsDTO {
	var userAuth entity.TUserAuth
	engine := ormInit.GetEngine()
	_, err := engine.Where("Username = ?", vo.Username).Get(&userAuth)
	if err != nil {
		zlog.Error(err.Error())
		return nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(userAuth.Password), []byte(vo.Password))
	if err != nil {
		return nil
	}
	var userInfo entity.TUserInfo
	_, err = engine.Where("id = ?", userAuth.UserInfoId).Get(&userInfo)
	if err != nil {
		zlog.Error(err.Error())
		return nil
	}
	var roles []string
	err = engine.SQL("select role_name from t_role where id in (select role_id from t_user_role where user_id = ?)", userInfo.Id).Find(&roles)
	if err != nil {
		zlog.Error(err.Error())
		return nil
	}
	userDetailsDTO := &model.UserDetailsDTO{
		Id:          userAuth.Id,
		UserInfoId:  userInfo.Id,
		Email:       userInfo.Email,
		LoginType:   userAuth.LoginType,
		Username:    userAuth.Username,
		Password:    userAuth.Password,
		Roles:       roles,
		Nickname:    userInfo.Nickname,
		Avatar:      userInfo.Avatar,
		Intro:       userInfo.Intro,
		Website:     userInfo.Website,
		IsSubscribe: userInfo.IsSubscribe,
		IsDisable:   userInfo.IsDisable,
	}
	return userDetailsDTO
}

func (u *MyUserAuthService) UpdateUserIp(user entity.TUserAuth) {
	if user.IpSource == "0" || user.IpSource == "" {
		user.IpSource = shared.UNKNOWN
	}
	session := ormInit.GetEngine().NewSession()
	err := session.Begin()
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	defer session.Close()

	_, err = session.Prepare().ID(user.Id).MustCols("ip_source", "ip_address").Update(&user)
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
		return
	}
	err = session.Commit()
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
		return
	}
}
