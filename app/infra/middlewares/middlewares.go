package middlewares

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	model2 "github.com/casbin/casbin/v2/model"
	xormadapter "github.com/casbin/xorm-adapter/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type bodyLog struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLog) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// 缓存 swagger.json 内容
var swaggerCache map[string]interface{}

// 配置项
const (
	SwaggerFilePath = "docs/swagger.json"
)

func init() {
	// 初始化 swagger 缓存
	swaggerCache = make(map[string]interface{})
	open, err := os.Open(SwaggerFilePath)
	if err != nil {
		zlog.Error("Failed to open swagger.json: " + err.Error())
		return
	}
	defer open.Close()

	bys, err := io.ReadAll(open)
	if err != nil {
		zlog.Error("Failed to read swagger.json: " + err.Error())
		return
	}

	err = json.Unmarshal(bys, &swaggerCache)
	if err != nil {
		zlog.Error("Failed to unmarshal swagger.json: " + err.Error())
		return
	}
	zlog.Info("Swagger cache initialized successfully")
}

// getSwaggerInfo 从缓存中获取 swagger 信息
func getSwaggerInfo(reqURI, reqMethod string) (module, desc string) {
	// 使用缓存的 swagger 内容
	hm := swaggerCache
	if hm == nil {
		zlog.Error("Swagger cache not initialized")
		return "Unknown", "Unknown"
	}

	apis, ok := hm["paths"].(map[string]interface{})
	if !ok {
		zlog.Error("Invalid swagger format: paths not found")
		return "Unknown", "Unknown"
	}

	pathData, ok := apis[reqURI].(map[string]interface{})
	if !ok {
		zlog.Error("Invalid swagger format: path not found")
		return "Unknown", "Unknown"
	}

	reqMethodData, ok := pathData[strings.ToLower(reqMethod)].(map[string]interface{})
	if !ok {
		zlog.Error("Invalid swagger format: method not found")
		return "Unknown", "Unknown"
	}

	module, ok = reqMethodData["summary"].(string)
	if !ok {
		module = "Unknown"
	}

	desc, ok = reqMethodData["description"].(string)
	if !ok {
		desc = "Unknown"
	}

	return module, desc
}

func Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		blw := &bodyLog{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		// 创建一个缓冲区
		var buf bytes.Buffer

		// 拷贝数据到缓冲区
		_, err := io.Copy(&buf, c.Request.Body)
		if err != nil {
			zlog.Error(err.Error())
		}

		// 获取缓冲区中的数据
		reqData := buf.Bytes()

		// 创建一个新的io.ReadCloser
		c.Request.Body = io.NopCloser(bytes.NewReader(reqData))

		c.Next()
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete {
			reqURI := strings.Split(c.Request.RequestURI, "?")[0]
			reqMethod := c.Request.Method
			ip := shared.GetIpAddress(c.Request)
			ipSource := shared.GetIpSource(ip)

			value, ok := c.Get("userInfo")
			var dto model.UserDetailsDTO
			if !ok {
				dto = model.UserDetailsDTO{
					Nickname:   "访客",
					UserInfoId: -1,
				}
			} else {
				dto = value.(model.UserDetailsDTO)
			}
			nickname := dto.Nickname
			userId := dto.UserInfoId

			optFunc := c.HandlerName()
			// 获取 swagger 信息
			module, desc := getSwaggerInfo(reqURI, reqMethod)
			optType := ""
			if strings.Contains(desc, "上传") {
				optType = "上传"
			} else if reqMethod == http.MethodPost {
				optType = "新增或修改"
			} else if reqMethod == http.MethodPut {
				optType = "修改"
			} else {
				optType = "删除"
			}
			resData := blw.body.String()
			optLog := entity.TOperationLog{
				OptModule:     module,
				OptType:       optType,
				OptUri:        reqURI,
				OptMethod:     optFunc,
				OptDesc:       desc,
				RequestParam:  string(reqData),
				RequestMethod: reqMethod,
				ResponseData:  resData,
				UserId:        userId,
				Nickname:      nickname,
				IpAddress:     ip,
				IpSource:      ipSource,
			}
			repository.EnqueueOptLog(optLog)
		}
		resData := make(map[string]interface{})
		json.Unmarshal(blw.body.Bytes(), &resData)
		if resData["code"] == 51000 {
			reqURI := strings.Split(c.Request.RequestURI, "?")[0]
			reqMethod := c.Request.Method
			ip := shared.GetIpAddress(c.Request)
			ipSource := shared.GetIpSource(ip)
			optFunc := c.HandlerName()

			// 获取 swagger 信息
			_, desc := getSwaggerInfo(reqURI, reqMethod)

			exLog := entity.TExceptionLog{
				OptUri:        reqURI,
				OptMethod:     optFunc,
				RequestMethod: reqMethod,
				RequestParam:  string(reqData),
				OptDesc:       desc,
				ExceptionInfo: "",
				IpAddress:     ip,
				IpSource:      ipSource,
			}
			repository.EnqueueExLog(exLog)
		}
	}
}

func SpiderReject() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shared.IsBot(c.Request) {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ResultFailWithMessage("You may be a robot！"))
			return
		}
		c.Next()
	}
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin != "" && config.IsAllowedOrigin(origin)
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		if c.Request.Method == http.MethodOptions {
			if origin != "" && !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func LoginFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request
		uri := req.URL.Path
		if uri == "/users/login" && req.Method == http.MethodPost {
			resultVO := Users(c)
			c.AbortWithStatusJSON(http.StatusOK, resultVO)
			return
		}
		c.Next()
	}
}

func Users(c *gin.Context) model.ResultVO {
	var userVO model.UserVO
	err := c.ShouldBind(&userVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage("登录失败，请联系管理员")
	}
	if !shared.CheckEmail(userVO.Username) {
		return model.ResultFailWithMessage("邮箱格式不正确！")
	}
	ctx := c.Request.Context()
	ipAddress := shared.GetIpAddress(c.Request)
	allowed, err := loginAttemptAllowed(ctx, userVO.Username, ipAddress)
	if err != nil {
		zlog.Error("login rate limiter failed: " + err.Error())
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if !allowed {
		return model.ResultFailWithMessage("账号或密码不正确！")
	}
	userDetailsDTO := userAuthService.CheckUserAuth(ctx, userVO)
	if userDetailsDTO == nil {
		if err := recordLoginFailure(ctx, userVO.Username, ipAddress); err != nil {
			zlog.Error("record login failure: " + err.Error())
		}
		return model.ResultFailWithMessage("账号或密码不正确！")
	}
	if err := clearLoginFailures(ctx, userVO.Username, ipAddress); err != nil {
		zlog.Error("clear login failures: " + err.Error())
	}
	region := shared.GetIpSource(ipAddress)
	userAuth := entity.TUserAuth{
		Id:            userDetailsDTO.Id,
		IpAddress:     ipAddress,
		IpSource:      region,
		LastLoginTime: time.Now(),
	}
	userAuthService.UpdateUserIp(ctx, userAuth)

	regionlist := strings.Split(region, "|")
	ipSource := "Unknown"
	if len(regionlist) >= 4 {
		ipSource = regionlist[2] + "|" + regionlist[3]
	}
	userDetailsDTO.IpAddress = ipAddress
	userDetailsDTO.IpSource = ipSource
	name, version := shared.GetBrowser(c.Request)
	osName := shared.GetOS(c.Request)
	userDetailsDTO.Browser = name + version
	userDetailsDTO.Os = osName

	accessToken, _, err := shared.CreateTokenCtx(ctx, userDetailsDTO)
	if err != nil {
		zlog.Error("Failed to create token: " + err.Error())
		return model.ResultFailWithMessage("登录失败")
	}
	var userInfoDTO model.UserInfoDTO
	marshal, err := json.Marshal(userDetailsDTO)
	if err != nil {
		zlog.Error(err.Error())
	}

	err = json.Unmarshal(marshal, &userInfoDTO)
	if err != nil {
		zlog.Error(err.Error())
	}
	userInfoDTO.Token = accessToken

	return model.ResultOkWithData(userInfoDTO)
}

const (
	loginFailureWindow = 10 * time.Minute
	loginLockDuration  = 15 * time.Minute
	loginAccountLimit  = int64(5)
	loginIPLimit       = int64(20)
)

func loginKey(prefix, value string) string {
	sum := sha256.Sum256([]byte(value))
	return prefix + hex.EncodeToString(sum[:])
}

func normalizedLoginUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func loginAttemptAllowed(ctx context.Context, username, ip string) (bool, error) {
	accountLock, err := shared.GetCtx(ctx, loginKey("login:lock:account:", normalizedLoginUsername(username)))
	if err != nil {
		return false, err
	}
	if accountLock != "" {
		return false, nil
	}
	ipLock, err := shared.GetCtx(ctx, loginKey("login:lock:ip:", ip))
	if err != nil {
		return false, err
	}
	return ipLock == "", nil
}

func recordLoginFailure(ctx context.Context, username, ip string) error {
	accountKey := loginKey("login:failure:account:", normalizedLoginUsername(username))
	ipKey := loginKey("login:failure:ip:", ip)
	accountCount, err := shared.IncrExpireCtx(ctx, accountKey, loginFailureWindow)
	if err != nil {
		return err
	}
	ipCount, err := shared.IncrExpireCtx(ctx, ipKey, loginFailureWindow)
	if err != nil {
		return err
	}
	if accountCount >= loginAccountLimit {
		if err := shared.SetWithTimeCtx(ctx, loginKey("login:lock:account:", normalizedLoginUsername(username)), "1", loginLockDuration); err != nil {
			return err
		}
	}
	if ipCount >= loginIPLimit {
		if err := shared.SetWithTimeCtx(ctx, loginKey("login:lock:ip:", ip), "1", loginLockDuration); err != nil {
			return err
		}
	}
	return nil
}

func clearLoginFailures(ctx context.Context, username, ip string) error {
	accountKey := loginKey("login:failure:account:", normalizedLoginUsername(username))
	ipKey := loginKey("login:failure:ip:", ip)
	if err := shared.DelCtx(ctx, accountKey); err != nil {
		return err
	}
	return shared.DelCtx(ctx, ipKey)
}

func AuthorizationFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只对后台接口进行认证检查
		if !isAdminPath(c.Request.URL.Path) {
			// 前台接口不限制，直接放行
			c.Next()
			return
		}

		authorization := strings.TrimSpace(c.Request.Header.Get(shared.TOKEN_HEADER))
		parts := strings.Fields(authorization)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "null" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
			return
		}
		token := parts[1]
		hm, err := shared.TokenParseCtx(c.Request.Context(), token)
		if err != nil || hm == nil {
			if err != nil {
				zlog.Error("token validation failed: " + err.Error())
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
			return
		}
		userAuthID, ok := hm["sub"].(string)
		if !ok || userAuthID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
			return
		}
		var userDetailsDTO model.UserDetailsDTO
		dto, err := shared.HGetCtx(c.Request.Context(), shared.LOGIN_USER, userAuthID)
		if err != nil {
			zlog.Error("load login user: " + err.Error())
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, model.ResultFailWithMessage("服务暂不可用"))
			return
		}
		if dto == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithCode(41000))
			return
		}
		if err = json.Unmarshal([]byte(dto), &userDetailsDTO); err != nil {
			zlog.Error(err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, model.ResultFailWithMessage("server error"))
			return
		}
		userDetailsDTO.LastLoginTime = time.Now()
		c.Set("userInfo", userDetailsDTO)
		c.Next()
	}
}

func isAdminPath(path string) bool {
	return path == "/admin" || strings.HasPrefix(path, "/admin/")
}

// getRolesByUserInfoId 获取用户角色，优先从缓存中获取
func getRolesByUserInfoId(ctx context.Context, userInfoId int) []string {
	cacheKey := fmt.Sprintf("roles:%d", userInfoId)
	cachedRoles, err := shared.HGetCtx(ctx, "user_roles", cacheKey)
	if err != nil {
		zlog.Error("load cached roles: " + err.Error())
	}
	if cachedRoles != "" {
		var roles []string
		if err := json.Unmarshal([]byte(cachedRoles), &roles); err == nil {
			return roles
		}
	}

	roles := roleRepo.ListRolesByUserInfoId(userInfoId)
	rolesJson, _ := json.Marshal(roles)
	if err := shared.HSetCtx(ctx, "user_roles", cacheKey, rolesJson, 1*time.Hour); err != nil {
		zlog.Error("cache roles: " + err.Error())
	}
	return roles
}

// checkPermission 检查用户是否有权限访问资源
func checkPermission(roles []string, uri, method string) (bool, string) {
	for _, role := range roles {
		ok, err := casbinEnforcer().Enforce(role, uri, method)
		if err != nil {
			errorMsg := "Permission check failed: " + err.Error()
			zlog.Error(errorMsg)
			return false, errorMsg
		}
		if ok {
			return true, ""
		}
	}
	return false, fmt.Sprintf("用户角色 %v 没有权限访问 %s %s", roles, method, uri)
}

func CasbinResourceFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAdminPath(c.Request.URL.Path) {
			value, ok := c.Get("userInfo")
			if !ok {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage("权限不足"))
				return
			}
			dto := value.(model.UserDetailsDTO)
			roles := getRolesByUserInfoId(c.Request.Context(), dto.UserInfoId)
			ok, errorMsg := checkPermission(roles, c.Request.URL.Path, c.Request.Method)
			if errorMsg != "" {
				if strings.Contains(errorMsg, "Permission check failed") {
					c.AbortWithStatusJSON(http.StatusInternalServerError, model.ResultFailWithMessage("权限检查失败"))
				} else {
					c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage(errorMsg))
				}
				return
			}
			if !ok {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage("权限不足"))
				return
			}
		}
		c.Next()
	}
}

// 配置项
const (
	IpLimiterMaxSize = 10000 // ipLimiter map 的最大容量
)

var (
	globalLimiter   = rate.NewLimiter(rate.Every(time.Second/20), 1200)
	ipLimiter       = make(map[string]*rate.Limiter)
	mutex           sync.Mutex
	roleRepo        repository.RoleRepo     = new(repository.MyRoleRepo)
	userAuthService service.UserAuthService = new(service.MyUserAuthService)
)

func AccessLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get("userInfo")
		if ok {
			dto := value.(model.UserDetailsDTO)
			roles := getRolesByUserInfoId(c.Request.Context(), dto.UserInfoId)
			ok, errorMsg := checkPermission(roles, c.Request.URL.Path, c.Request.Method)
			if errorMsg != "" {
				if strings.Contains(errorMsg, "Permission check failed") {
					zlog.Error("Permission check failed in AccessLimiter: " + errorMsg)
				}
				// 权限检查失败，继续进行速率限制
			} else if ok {
				c.Next()
				return
			}
		}

		ip := shared.GetIpAddress(c.Request)
		mutex.Lock()
		limiter, ok := ipLimiter[ip]
		if !ok {
			// 检查 ipLimiter map 的大小是否超过了最大容量
			if len(ipLimiter) >= IpLimiterMaxSize {
				// 清理一部分旧的条目，保留 80% 的容量
				cleanupSize := len(ipLimiter) - (IpLimiterMaxSize * 80 / 100)
				count := 0
				for key := range ipLimiter {
					delete(ipLimiter, key)
					count++
					if count >= cleanupSize {
						break
					}
				}
				zlog.Info(fmt.Sprintf("Cleaned up %d old IP limiters", count))
			}
			limiter = rate.NewLimiter(rate.Every(time.Minute/60), 60)
			ipLimiter[ip] = limiter
		}
		mutex.Unlock()

		if !limiter.Allow() || !globalLimiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.ResultFailWithMessage("请求过于频繁"))
			return
		}
		c.Next()
	}
}

var (
	adapterOnce  sync.Once
	enforcerOnce sync.Once
	xormAdapter  *xormadapter.Adapter
	enforcer     *casbin.Enforcer
)

const (
	text = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && r.act == p.act
`
)

// casbinAdapter 外部存储，如果需要的话
func casbinAdapter() *xormadapter.Adapter {
	adapterOnce.Do(func() {
		a, err := xormadapter.NewAdapterByEngine(ormInit.GetEngine())
		if err != nil {
			zlog.Fatal(err.Error())
		}
		xormAdapter = a
	})
	return xormAdapter
}

func casbinEnforcer() *casbin.Enforcer {
	enforcerOnce.Do(func() {
		m, err := model2.NewModelFromString(text)
		if err != nil {
			zlog.Fatal(err.Error())
		}

		e, err := casbin.NewEnforcer(m, casbinAdapter())
		if err != nil {
			zlog.Fatal(err.Error())
		}

		err = e.LoadPolicy()
		if err != nil {
			zlog.Fatal("策略未成功加载", zap.Error(err))
		}
		enforcer = e
	})

	return enforcer
}
