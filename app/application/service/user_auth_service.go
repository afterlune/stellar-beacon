package service

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"golang.org/x/crypto/bcrypt"
)

type UserAuthService interface {
	SendCode(c *gin.Context) model.ResultVO
	ListUserAreas(c *gin.Context) model.ResultVO
	ListUsers(c *gin.Context) model.ResultVO
	Register(c *gin.Context) model.ResultVO
	UpdatePassword(c *gin.Context) model.ResultVO
	UpdateAdminPassword(c *gin.Context) model.ResultVO
	Logout(ctx context.Context, id int) model.ResultVO
	CheckUser(ctx context.Context, vo model.UserVO) bool
	CheckUserAuth(ctx context.Context, vo model.UserVO) *model.UserDetailsDTO
	Authenticate(ctx context.Context, vo model.UserVO) (*model.UserDetailsDTO, error)
	UpdateUserIp(ctx context.Context, user entity.TUserAuth) error
}

type MyUserAuthService struct {
	repo    port.AuthRepository
	website BenetnaschInfoService
	cache   port.Cache
	mailer  port.Mailer
	visitor port.VisitorResolver
}

func NewUserAuthService(deps UserAuthServiceDeps) (*MyUserAuthService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyUserAuthService{
		repo:    deps.Repo,
		website: deps.Website,
		cache:   deps.Cache,
		mailer:  deps.Mailer,
		visitor: deps.Visitor,
	}, nil
}

func (u *MyUserAuthService) authRepository() port.AuthRepository {
	return u.repo
}

func (u *MyUserAuthService) websiteService() BenetnaschInfoService {
	return u.website
}

func (u *MyUserAuthService) SendCode(c *gin.Context) model.ResultVO {
	username := strings.ToLower(strings.TrimSpace(c.Query("username")))
	if !support.CheckEmail(username) {
		return model.ResultFailWithMessage("请输入正确邮箱")
	}
	ctx := c.Request.Context()
	if u.cache == nil || u.mailer == nil || u.visitor == nil {
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	key := func(prefix, value string) string {
		sum := sha256.Sum256([]byte(value))
		return prefix + hex.EncodeToString(sum[:])
	}
	if allowed, err := u.cache.SetNX(ctx, key("captcha:cooldown:", username), "1", time.Minute); err != nil {
		slog.Error("captcha rate limiter failed", "error", err)
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	} else if !allowed {
		return model.ResultFailWithMessage("验证码发送过于频繁")
	}
	emailCount, err := u.cache.IncrementWithExpiry(ctx, key("captcha:email:", username), time.Hour)
	if err != nil {
		slog.Error("captcha email limiter failed", "error", err)
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	identity, err := u.visitor.Resolve(ctx, c.Request)
	if err != nil {
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	ipCount, err := u.cache.IncrementWithExpiry(ctx, key("captcha:ip:", identity.IP), time.Hour)
	if err != nil {
		slog.Error("captcha IP limiter failed", "error", err)
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if emailCount > 5 || ipCount > 20 {
		return model.ResultFailWithMessage("验证码发送过于频繁")
	}
	code := support.RandomCode()
	codem := make(map[string]interface{})
	codem["content"] = "您的验证码为 " + code + " 有效期15分钟，请不要告诉他人哦！"
	emailDTO := model.EmailDTO{
		Email:      username,
		CommentMap: codem,
		Template:   "resource/template/common.html",
		Subject:    support.Captcha,
	}
	if err := u.mailer.SendHTML(ctx, port.EmailMessage{To: emailDTO.Email, Subject: emailDTO.Subject, Template: emailDTO.Template, CommentMap: emailDTO.CommentMap}); err != nil {
		slog.Error("send captcha email failed", "error", err)
		return model.ResultFailWithMessage("验证码发送失败")
	}
	if err := u.cache.Set(ctx, support.UserCodeKey+username, code, 15*time.Minute); err != nil {
		slog.Error("store captcha failed", "error", err)
		return model.ResultFailWithMessage("验证码发送失败")
	}
	return model.ResultOk()
}

func (u *MyUserAuthService) ListUserAreas(c *gin.Context) model.ResultVO {
	typeId := c.Query("type")
	var userAreaDTOs []model.UserAreaDTO
	switch typeId {
	case "1":
		if u.cache == nil {
			return model.ResultFail()
		}
		userArea, err := u.cache.Get(c.Request.Context(), support.UserArea)
		if err != nil {
			slog.Error("load user area failed", "error", err)
			return model.ResultFail()
		}
		if userArea != "" {
			err := json.Unmarshal([]byte(userArea), &userAreaDTOs)
			if err != nil {
				slog.Error("decode user area failed", "error", err)
			}
		}
		return model.ResultOkWithData(userAreaDTOs)
	case "2":
		if u.cache == nil {
			return model.ResultFail()
		}
		visitorArea, err := u.cache.HGetAll(c.Request.Context(), support.VisitorArea)
		if err != nil {
			slog.Error("load visitor area failed", "error", err)
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
				slog.Error("parse visitor count failed", "error", err)
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
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	users, count, err := u.authRepository().ListUsers(c.Request.Context(), port.UserFilter{
		Current:   vo.Current,
		Size:      vo.Size,
		Keywords:  vo.Keywords,
		LoginType: vo.LonginType,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: users, Count: int(count)})
}

func (u *MyUserAuthService) Register(c *gin.Context) model.ResultVO {
	var userVo model.UserVO
	if err := c.ShouldBind(&userVo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}

	if u.cache == nil {
		return model.ResultFail()
	}
	code, err := u.cache.Get(c.Request.Context(), support.UserCodeKey+strings.ToLower(strings.TrimSpace(userVo.Username)))
	if err != nil {
		slog.Error("load registration captcha failed", "error", err)
		return model.ResultFail()
	}
	if userVo.Code != code {
		return model.ResultFailWithMessage("验证码有误！")
	}

	if !support.CheckEmail(userVo.Username) {
		return model.ResultFailWithMessage("邮箱格式不对！")
	}
	username := strings.ToLower(strings.TrimSpace(userVo.Username))
	userVo.Username = username
	exists, err := u.userExists(c.Request.Context(), username)
	if err != nil {
		return model.ResultFromError(err)
	}
	if exists {
		return model.ResultFailWithMessage("邮箱已被注册！")
	}
	websiteConfigResult := u.websiteService().GetWebsiteConfig(c.Request.Context())
	websiteConfig, ok := websiteConfigResult.Data.(model.WebsiteConfigDTO)
	if !ok {
		return websiteConfigResult
	}
	userInfo := entity.TUserInfo{
		Email:    username,
		Nickname: support.DefaultNickname,
		Avatar:   websiteConfig.UserAvatar,
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userVo.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash registration password failed", "error", err)
		return model.ResultFailWithMessage("注册失败，稍后再试")
	}
	userAuth := entity.TUserAuth{
		Username:  username,
		Password:  string(password),
		LoginType: 1,
	}
	if err := u.authRepository().CreateUser(c.Request.Context(), userInfo, userAuth, 2); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (u *MyUserAuthService) UpdatePassword(c *gin.Context) model.ResultVO {
	var userVO model.UserVO
	if err := c.ShouldBind(&userVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	userVO.Username = strings.ToLower(strings.TrimSpace(userVO.Username))
	if u.cache == nil {
		return model.ResultFail()
	}
	code, err := u.cache.Get(c.Request.Context(), support.UserCodeKey+strings.ToLower(strings.TrimSpace(userVO.Username)))
	if err != nil {
		slog.Error("load password reset captcha failed", "error", err)
		return model.ResultFail()
	}
	if userVO.Code != code {
		return model.ResultFailWithMessage("验证码有误！")
	}
	if !support.CheckEmail(userVO.Username) {
		return model.ResultFailWithMessage("邮箱格式不对！")
	}
	exists, err := u.userExists(c.Request.Context(), userVO.Username)
	if err != nil {
		return model.ResultFromError(err)
	}
	if !exists {
		return model.ResultFailWithMessage("邮箱未注册！")
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userVO.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash password reset password failed", "error", err)
		return model.ResultFail()
	}
	if err := u.authRepository().UpdatePassword(c.Request.Context(), userVO.Username, string(password)); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (u *MyUserAuthService) UpdateAdminPassword(c *gin.Context) model.ResultVO {
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "auth.update_admin_password", nil))
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "auth.update_admin_password", nil))
	}

	var passwordVO model.PasswordVO
	err := c.ShouldBind(&passwordVO)
	if err != nil || passwordVO.NewPassword == "" || passwordVO.OldPassword == "" {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	user, err := u.authRepository().FindByID(c.Request.Context(), dto.Id)
	if err != nil {
		return model.ResultFromError(err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(passwordVO.OldPassword))
	if err != nil {
		return model.ResultFailWithMessage("旧密码不正确")
	}
	password, err := bcrypt.GenerateFromPassword([]byte(passwordVO.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return model.ResultFromError(err)
	}
	if user.Username != "" && err == nil {
		if err := u.authRepository().UpdatePasswordByID(c.Request.Context(), dto.Id, string(password)); err != nil {
			return model.ResultFromError(err)
		}
		return model.ResultOk()
	}
	return model.ResultFailWithMessage("旧密码不正确")
}

func (u *MyUserAuthService) Logout(ctx context.Context, id int) model.ResultVO {
	if u.cache == nil {
		return model.ResultFailWithMessage("注销失败，请稍后再试")
	}
	if err := u.cache.HDel(ctx, support.LoginUser, strconv.Itoa(id)); err != nil {
		slog.Error("remove login session failed", "error", err)
		return model.ResultFailWithMessage("注销失败，请稍后再试")
	}
	return model.ResultOkWithData(model.UserLogoutStatusDTO{
		Message: "注销成功",
	})
}

func (u *MyUserAuthService) userExists(ctx context.Context, username string) (bool, error) {
	_, err := u.authRepository().FindByUsername(ctx, username)
	if err == nil {
		return true, nil
	}
	if apperrors.IsKind(err, apperrors.KindNotFound) {
		return false, nil
	}
	return false, err
}

func (u *MyUserAuthService) CheckUser(ctx context.Context, vo model.UserVO) bool {
	exists, err := u.userExists(ctx, strings.ToLower(strings.TrimSpace(vo.Username)))
	if err != nil {
		slog.Error("check user failed", "error", err)
	}
	return exists
}

func (u *MyUserAuthService) Authenticate(ctx context.Context, vo model.UserVO) (*model.UserDetailsDTO, error) {
	username := strings.ToLower(strings.TrimSpace(vo.Username))
	user, err := u.authRepository().FindByUsername(ctx, username)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Auth.Password), []byte(vo.Password)); err != nil {
		return nil, nil
	}
	return &model.UserDetailsDTO{
		Id:          user.Auth.Id,
		UserInfoId:  user.Info.Id,
		Email:       user.Info.Email,
		LoginType:   user.Auth.LoginType,
		Username:    user.Auth.Username,
		Password:    user.Auth.Password,
		Roles:       user.Roles,
		Nickname:    user.Info.Nickname,
		Avatar:      user.Info.Avatar,
		Intro:       user.Info.Intro,
		Website:     user.Info.Website,
		IsSubscribe: user.Info.IsSubscribe,
		IsDisable:   user.Info.IsDisable,
	}, nil
}

func (u *MyUserAuthService) CheckUserAuth(ctx context.Context, vo model.UserVO) *model.UserDetailsDTO {
	dto, err := u.Authenticate(ctx, vo)
	if err != nil {
		slog.Error("authenticate user failed", "error", err)
		return nil
	}
	return dto
}

func (u *MyUserAuthService) UpdateUserIp(ctx context.Context, user entity.TUserAuth) error {
	if user.IpSource == "0" || user.IpSource == "" {
		user.IpSource = support.Unknown
	}
	return u.authRepository().UpdateLoginMetadata(ctx, user)
}
