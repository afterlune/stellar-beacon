package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"golang.org/x/crypto/bcrypt"
)

type UserAuthService interface {
	SendCode(c port.Request) port.ResultVO
	ListUserAreas(c port.Request) port.ResultVO
	ListUsers(c port.Request) port.ResultVO
	Register(c port.Request) port.ResultVO
	UpdatePassword(c port.Request) port.ResultVO
	UpdateAdminPassword(c port.Request) port.ResultVO
	Logout(ctx context.Context, id int) port.ResultVO
	CheckUser(ctx context.Context, vo port.UserVO) bool
	CheckUserAuth(ctx context.Context, vo port.UserVO) *port.UserDetailsDTO
	Authenticate(ctx context.Context, vo port.UserVO) (*port.UserDetailsDTO, error)
	UpdateUserIp(ctx context.Context, user port.TUserAuth) error
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

func (u *MyUserAuthService) SendCode(c port.Request) port.ResultVO {
	username := strings.ToLower(strings.TrimSpace(c.Query("username")))
	if !support.CheckEmail(username) {
		return port.ResultFailWithMessage("请输入正确邮箱")
	}
	ctx := c.Context()
	if u.cache == nil || u.mailer == nil || u.visitor == nil {
		return port.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	key := func(prefix, value string) string {
		sum := sha256.Sum256([]byte(value))
		return prefix + hex.EncodeToString(sum[:])
	}
	if allowed, err := u.cache.SetNX(ctx, key("captcha:cooldown:", username), "1", time.Minute); err != nil {
		slog.Error("captcha rate limiter failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("系统繁忙，请稍后再试")
	} else if !allowed {
		return port.ResultFailWithMessage("验证码发送过于频繁")
	}
	emailCount, err := u.cache.IncrementWithExpiry(ctx, key("captcha:email:", username), time.Hour)
	if err != nil {
		slog.Error("captcha email limiter failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	identity, err := u.visitor.Resolve(ctx, c.HTTPRequest())
	if err != nil {
		return port.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	ipCount, err := u.cache.IncrementWithExpiry(ctx, key("captcha:ip:", identity.IP), time.Hour)
	if err != nil {
		slog.Error("captcha IP limiter failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if emailCount > 5 || ipCount > 20 {
		return port.ResultFailWithMessage("验证码发送过于频繁")
	}
	code := support.RandomCode()
	codem := make(map[string]interface{})
	codem["content"] = "您的验证码为 " + code + " 有效期15分钟，请不要告诉他人哦！"
	emailDTO := port.EmailDTO{
		Email:      username,
		CommentMap: codem,
		Template:   "resource/template/common.html",
		Subject:    support.Captcha,
	}
	if err := u.mailer.SendHTML(ctx, port.EmailMessage{To: emailDTO.Email, Subject: emailDTO.Subject, Template: emailDTO.Template, CommentMap: emailDTO.CommentMap}); err != nil {
		slog.Error("send captcha email failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("验证码发送失败")
	}
	if err := u.cache.Set(ctx, support.UserCodeKey+username, code, 15*time.Minute); err != nil {
		slog.Error("store captcha failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("验证码发送失败")
	}
	return port.ResultOk()
}

func (u *MyUserAuthService) ListUserAreas(c port.Request) port.ResultVO {
	typeId := c.Query("type")
	var userAreaDTOs []port.UserAreaDTO
	switch typeId {
	case "1":
		if u.cache == nil {
			return port.ResultFail()
		}
		userArea, err := u.cache.Get(c.Context(), support.UserArea)
		if err != nil {
			slog.Error("load user area failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
		if userArea != "" {
			err := json.Unmarshal([]byte(userArea), &userAreaDTOs)
			if err != nil {
				slog.Error("decode user area failed", "error_code", apperrors.SafeCode(err))
				return port.ResultFail()
			}
		}
		return port.ResultOkWithData(userAreaDTOs)
	case "2":
		if u.cache == nil {
			return port.ResultFail()
		}
		visitorArea, err := u.cache.HGetAll(c.Context(), support.VisitorArea)
		if err != nil {
			slog.Error("load visitor area failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
		for k, v := range visitorArea {
			userAreaDTO, reason := parseVisitorArea(k, v)
			if reason != "" {
				slog.WarnContext(c.Context(), "skip malformed visitor area record", "reason", reason)
				continue
			}
			userAreaDTOs = append(userAreaDTOs, userAreaDTO)
		}
	default:
		break
	}
	return port.ResultOkWithData(userAreaDTOs)
}

func parseVisitorArea(field, value string) (port.UserAreaDTO, string) {
	parts := strings.Split(field, "|")
	if len(parts) < 3 {
		return port.UserAreaDTO{}, "invalid region format"
	}
	region := strings.TrimSpace(parts[2])
	if region == "" {
		return port.UserAreaDTO{}, "empty region"
	}
	visitCount, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || visitCount < 0 {
		return port.UserAreaDTO{}, "invalid visit count"
	}
	return port.UserAreaDTO{
		Name:  strings.TrimSuffix(region, "省"),
		Value: visitCount,
	}, ""
}

func (u *MyUserAuthService) ListUsers(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	loginType := vo.LoginType
	if loginType == 0 {
		loginType = vo.LonginType
	}
	users, count, err := u.authRepository().ListUsers(c.Context(), port.UserFilter{
		Current:   vo.Current,
		Size:      vo.Size,
		Keywords:  vo.Keywords,
		LoginType: loginType,
	})
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: users, Count: int(count)})
}

func (u *MyUserAuthService) Register(c port.Request) port.ResultVO {
	var userVo port.UserVO
	if err := c.Bind(&userVo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}

	if u.cache == nil {
		return port.ResultFail()
	}
	code, err := u.cache.Get(c.Context(), support.UserCodeKey+strings.ToLower(strings.TrimSpace(userVo.Username)))
	if err != nil {
		slog.Error("load registration captcha failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	if userVo.Code != code {
		return port.ResultFailWithMessage("验证码有误！")
	}

	if !support.CheckEmail(userVo.Username) {
		return port.ResultFailWithMessage("邮箱格式不对！")
	}
	username := strings.ToLower(strings.TrimSpace(userVo.Username))
	userVo.Username = username
	exists, err := u.userExists(c.Context(), username)
	if err != nil {
		return port.ResultFromError(err)
	}
	if exists {
		return port.ResultFailWithMessage("邮箱已被注册！")
	}
	websiteConfigResult := u.websiteService().GetWebsiteConfig(c.Context())
	websiteConfig, ok := websiteConfigResult.Data.(port.WebsiteConfigDTO)
	if !ok {
		return websiteConfigResult
	}
	userInfo := port.TUserInfo{
		Email:    username,
		Nickname: support.DefaultNickname,
		Avatar:   websiteConfig.UserAvatar,
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userVo.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash registration password failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("注册失败，稍后再试")
	}
	userAuth := port.TUserAuth{
		Username:  username,
		Password:  string(password),
		LoginType: 1,
	}
	if err := u.authRepository().CreateUser(c.Context(), userInfo, userAuth, 2); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (u *MyUserAuthService) UpdatePassword(c port.Request) port.ResultVO {
	var userVO port.UserVO
	if err := c.Bind(&userVO); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	userVO.Username = strings.ToLower(strings.TrimSpace(userVO.Username))
	if u.cache == nil {
		return port.ResultFail()
	}
	code, err := u.cache.Get(c.Context(), support.UserCodeKey+strings.ToLower(strings.TrimSpace(userVO.Username)))
	if err != nil {
		slog.Error("load password reset captcha failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	if userVO.Code != code {
		return port.ResultFailWithMessage("验证码有误！")
	}
	if !support.CheckEmail(userVO.Username) {
		return port.ResultFailWithMessage("邮箱格式不对！")
	}
	exists, err := u.userExists(c.Context(), userVO.Username)
	if err != nil {
		return port.ResultFromError(err)
	}
	if !exists {
		return port.ResultFailWithMessage("邮箱未注册！")
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userVO.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash password reset password failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	if err := u.authRepository().UpdatePassword(c.Context(), userVO.Username, string(password)); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (u *MyUserAuthService) UpdateAdminPassword(c port.Request) port.ResultVO {
	value, ok := c.Get("userInfo")
	if !ok {
		return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "auth.update_admin_password", nil))
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok {
		return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "auth.update_admin_password", nil))
	}

	var passwordVO port.PasswordVO
	err := c.Bind(&passwordVO)
	if err != nil || passwordVO.NewPassword == "" || passwordVO.OldPassword == "" {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	user, err := u.authRepository().FindByID(c.Context(), dto.Id)
	if err != nil {
		return port.ResultFromError(err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(passwordVO.OldPassword))
	if err != nil {
		return port.ResultFailWithMessage("旧密码不正确")
	}
	password, err := bcrypt.GenerateFromPassword([]byte(passwordVO.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return port.ResultFromError(err)
	}
	if user.Username != "" && err == nil {
		if err := u.authRepository().UpdatePasswordByID(c.Context(), dto.Id, string(password)); err != nil {
			return port.ResultFromError(err)
		}
		return port.ResultOk()
	}
	return port.ResultFailWithMessage("旧密码不正确")
}

func (u *MyUserAuthService) Logout(ctx context.Context, id int) port.ResultVO {
	if u.cache == nil {
		return port.ResultFailWithMessage("注销失败，请稍后再试")
	}
	if err := u.cache.HDel(ctx, support.LoginUser, strconv.Itoa(id)); err != nil {
		slog.Error("remove login session failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("注销失败，请稍后再试")
	}
	return port.ResultOkWithData(port.UserLogoutStatusDTO{
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

func (u *MyUserAuthService) CheckUser(ctx context.Context, vo port.UserVO) bool {
	exists, err := u.userExists(ctx, strings.ToLower(strings.TrimSpace(vo.Username)))
	if err != nil {
		slog.Error("check user failed", "error_code", apperrors.SafeCode(err))
	}
	return exists
}

func (u *MyUserAuthService) Authenticate(ctx context.Context, vo port.UserVO) (*port.UserDetailsDTO, error) {
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
	return &port.UserDetailsDTO{
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

func (u *MyUserAuthService) CheckUserAuth(ctx context.Context, vo port.UserVO) *port.UserDetailsDTO {
	dto, err := u.Authenticate(ctx, vo)
	if err != nil {
		slog.Error("authenticate user failed", "error_code", apperrors.SafeCode(err))
		return nil
	}
	return dto
}

func (u *MyUserAuthService) UpdateUserIp(ctx context.Context, user port.TUserAuth) error {
	if user.IpSource == "0" || user.IpSource == "" {
		user.IpSource = support.Unknown
	}
	return u.authRepository().UpdateLoginMetadata(ctx, user)
}
