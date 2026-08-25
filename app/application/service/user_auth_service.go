package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

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
	Logout(ctx context.Context, id int) model.ResultVO
	QQLogin(c *gin.Context) model.ResultVO
	CheckUser(ctx context.Context, vo model.UserVO) bool
	CheckUserAuth(ctx context.Context, vo model.UserVO) *model.UserDetailsDTO
	UpdateUserIp(ctx context.Context, user entity.TUserAuth)
}

type MyUserAuthService struct{}

func (u *MyUserAuthService) SendCode(c *gin.Context) model.ResultVO {
	username := strings.ToLower(strings.TrimSpace(c.Query("username")))
	if !shared.CheckEmail(username) {
		return model.ResultFailWithMessage("请输入正确邮箱")
	}
	ctx := c.Request.Context()
	key := func(prefix, value string) string {
		sum := sha256.Sum256([]byte(value))
		return prefix + hex.EncodeToString(sum[:])
	}
	if allowed, err := shared.SetNXCtx(ctx, key("captcha:cooldown:", username), "1", time.Minute); err != nil {
		zlog.Error("captcha rate limiter failed: " + err.Error())
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	} else if !allowed {
		return model.ResultFailWithMessage("验证码发送过于频繁")
	}
	emailCount, err := shared.IncrExpireCtx(ctx, key("captcha:email:", username), time.Hour)
	if err != nil {
		zlog.Error("captcha email limiter failed: " + err.Error())
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	ipCount, err := shared.IncrExpireCtx(ctx, key("captcha:ip:", shared.GetIpAddress(c.Request)), time.Hour)
	if err != nil {
		zlog.Error("captcha IP limiter failed: " + err.Error())
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if emailCount > 5 || ipCount > 20 {
		return model.ResultFailWithMessage("验证码发送过于频繁")
	}
	code := shared.RandomCode()
	codem := make(map[string]interface{})
	codem["content"] = "您的验证码为 " + code + " 有效期15分钟，请不要告诉他人哦！"
	emailDTO := model.EmailDTO{
		Email:      username,
		CommentMap: codem,
		Template:   "resource/template/common.html",
		Subject:    shared.CAPTCHA,
	}
	if err := shared.SendHtmlEmail(emailDTO); err != nil {
		zlog.Error("send captcha email failed: " + err.Error())
		return model.ResultFailWithMessage("验证码发送失败")
	}
	if err := shared.SetWithTimeCtx(ctx, shared.USER_CODE_KEY+username, code, 15*time.Minute); err != nil {
		zlog.Error("store captcha: " + err.Error())
		return model.ResultFailWithMessage("验证码发送失败")
	}
	return model.ResultOk()
}

func (u *MyUserAuthService) ListUserAreas(c *gin.Context) model.ResultVO {
	typeId := c.Query("type")
	var userAreaDTOs []model.UserAreaDTO
	switch typeId {
	case "1":
		userArea, err := shared.GetCtx(c.Request.Context(), shared.USER_AREA)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		if userArea != "" {
			err := json.Unmarshal([]byte(userArea), &userAreaDTOs)
			if err != nil {
				zlog.Error(err.Error())
			}
		}
		return model.ResultOkWithData(userAreaDTOs)
	case "2":
		visitorArea, err := shared.HGetAllCtx(c.Request.Context(), shared.VISITOR_AREA)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
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

	code, err := shared.GetCtx(c.Request.Context(), shared.USER_CODE_KEY+strings.ToLower(strings.TrimSpace(userVo.Username)))
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if userVo.Code != code {
		return model.ResultFailWithMessage("验证码有误！")
	}

	if !shared.CheckEmail(userVo.Username) {
		return model.ResultFailWithMessage("邮箱格式不对！")
	}
	username := strings.ToLower(strings.TrimSpace(userVo.Username))
	userVo.Username = username
	if u.CheckUser(c.Request.Context(), userVo) {
		return model.ResultFailWithMessage("邮箱已被注册！")
	}
	userInfo := entity.TUserInfo{
		Email:    username,
		Nickname: shared.DEFAULT_NICKNAME,
		Avatar:   benetnaschService.GetWebsiteConfig(c.Request.Context()).Data.(model.WebsiteConfigDTO).UserAvatar,
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userVo.Password), bcrypt.DefaultCost)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}
	userAuth := entity.TUserAuth{
		Username:  username,
		Password:  string(password),
		LoginType: 1,
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		if _, err := session.Insert(&userInfo); err != nil {
			return err
		}
		userRole := entity.TUserRole{UserId: userInfo.Id, RoleId: 2}
		if _, err := session.Insert(&userRole); err != nil {
			return err
		}
		userAuth.UserInfoId = userInfo.Id
		_, err := session.Insert(&userAuth)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}
	return model.ResultOk()
}

func (u *MyUserAuthService) UpdatePassword(c *gin.Context) model.ResultVO {
	var userVO model.UserVO
	err := c.ShouldBind(&userVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	userVO.Username = strings.ToLower(strings.TrimSpace(userVO.Username))
	code, err := shared.GetCtx(c.Request.Context(), shared.USER_CODE_KEY+strings.ToLower(strings.TrimSpace(userVO.Username)))
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if userVO.Code != code {
		return model.ResultFailWithMessage("验证码有误！")
	}
	if !shared.CheckEmail(userVO.Username) {
		return model.ResultFailWithMessage("邮箱格式不对！")
	}
	if !u.CheckUser(c.Request.Context(), userVO) {
		return model.ResultFailWithMessage("邮箱未注册！")
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userVO.Password), bcrypt.DefaultCost)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	userAuth := entity.TUserAuth{
		Password: string(password),
		Username: userVO.Username,
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Where("username = ?", userAuth.Username).Update(&userAuth)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage(err.Error())
	}
	return model.ResultOk()
}

func (u *MyUserAuthService) UpdateAdminPassword(c *gin.Context) model.ResultVO {
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	var passwordVO model.PasswordVO
	err := c.ShouldBind(&passwordVO)
	if err != nil || passwordVO.NewPassword == "" || passwordVO.OldPassword == "" {
		if err != nil {
			zlog.Error(err.Error())
		}
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
		if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
			_, err := session.ID(userAuth.Id).Update(&userAuth)
			return err
		}); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		return model.ResultOk()
	}
	return model.ResultFailWithMessage("旧密码不正确")
}

func (u *MyUserAuthService) Logout(ctx context.Context, id int) model.ResultVO {
	if err := shared.HDelCtx(ctx, shared.LOGIN_USER, strconv.Itoa(id)); err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage("注销失败，请稍后再试")
	}
	return model.ResultOkWithData(model.UserLogoutStatusDTO{
		Message: "注销成功",
	})
}

func (u *MyUserAuthService) QQLogin(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (u *MyUserAuthService) CheckUser(ctx context.Context, vo model.UserVO) bool {
	var userAuth entity.TUserAuth
	b, err := ormInit.GetEngine().Context(ctx).Where("Username = ?", vo.Username).Get(&userAuth)
	if err != nil {
		zlog.Error(err.Error())
		return false
	}
	return b
}

func (u *MyUserAuthService) CheckUserAuth(ctx context.Context, vo model.UserVO) *model.UserDetailsDTO {
	var userAuth entity.TUserAuth
	engine := ormInit.GetEngine().Context(ctx)
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

func (u *MyUserAuthService) UpdateUserIp(ctx context.Context, user entity.TUserAuth) {
	if user.IpSource == "0" || user.IpSource == "" {
		user.IpSource = shared.UNKNOWN
	}
	if err := ormInit.WithTx(ctx, func(session *xorm.Session) error {
		_, err := session.ID(user.Id).MustCols("ip_source", "ip_address").Update(&user)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return
	}
}
