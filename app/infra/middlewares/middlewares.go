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
			go repository.SaveOptLog(optLog)
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
			go repository.SaveExLog(exLog)
		}
	}
}

func SpiderReject() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shared.IsBot(c.Request) || !strings.Contains(c.Request.Host, config.Verification) ||
			!strings.Contains(c.Request.Referer(), config.Verification) {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ResultFailWithMessage("You may be a robot！"))
			return
		}
		c.Next()
	}
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		// 限制 CORS 来源，使用配置中的 verification 值
		c.Header("Access-Control-Allow-Origin", "http://"+config.Verification)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func LoginFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request
		uri := req.RequestURI
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
	userDetailsDTO := userAuthService.CheckUserAuth(userVO)
	if userDetailsDTO == nil {
		return model.ResultFailWithMessage("账号或密码不正确！")
	}
	ipAddress := shared.GetIpAddress(c.Request)
	region := shared.GetIpSource(ipAddress)
	userAuth := entity.TUserAuth{
		Id:            userDetailsDTO.Id,
		IpAddress:     ipAddress,
		IpSource:      region,
		LastLoginTime: time.Now(),
	}
	userAuthService.UpdateUserIp(userAuth)

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

	token := shared.CreateToken(userDetailsDTO)
	var userInfoDTO model.UserInfoDTO
	marshal, err := json.Marshal(userDetailsDTO)
	if err != nil {
		zlog.Error(err.Error())
	}

	err = json.Unmarshal(marshal, &userInfoDTO)
	if err != nil {
		zlog.Error(err.Error())
	}
	userInfoDTO.Token = token

	return model.ResultOkWithData(userInfoDTO)
}

func AuthorizationFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只对后台接口进行认证检查
		if strings.Contains(c.Request.RequestURI, "/admin") {
			authorization := c.Request.Header.Get(shared.TOKEN_HEADER)
			if strings.Contains(authorization, shared.TOKEN_PREFIX) {
				strs := strings.Split(authorization, shared.TOKEN_PREFIX)
				if len(strs) == 2 {
					token := strs[1]
					if token == "" || token == "null" {
						c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
						return
					}
					hm := shared.TokenParse(token)
					if hm == nil {
						c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
						return
					}
					userAuthId, ok := hm["sub"].(string)
					if !ok || userAuthId == "" {
						c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
						return
					}
					var userDetailsDTO model.UserDetailsDTO

					dto := shared.HGet(shared.LOGIN_USER, userAuthId)
					if dto == "" {
						c.Abort()
						c.JSON(http.StatusOK, model.ResultFailWithCode(41000))
						return
					}
					err := json.Unmarshal([]byte(dto), &userDetailsDTO)
					if err != nil {
						zlog.Error(err.Error())
						c.Abort()
						c.JSON(http.StatusInternalServerError, model.ResultFailWithMessage("server error"))
						return
					}
					userDetailsDTO.LastLoginTime = time.Now()
					c.Set("userInfo", userDetailsDTO)
					c.Next()
				}
			}
			if !strings.Contains(authorization, shared.TOKEN_PREFIX) {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage("非法操作"))
				return
			}
		} else {
			// 前台接口不限制，直接放行
			c.Next()
		}
	}
}

func CasbinResourceFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.Contains(c.Request.RequestURI, "/admin") {
			value, ok := c.Get("userInfo")
			if !ok {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage("权限不足"))
				return
			}
			dto := value.(model.UserDetailsDTO)
			roles := roleRepo.ListRolesByUserInfoId(dto.UserInfoId)
			var flag bool
			for _, role := range roles {
				ok, _ := casbinEnforcer().Enforce(role, c.Request.RequestURI, c.Request.Method)
				if ok {
					flag = true
					break
				}
			}
			if !flag {
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
			roles := roleRepo.ListRolesByUserInfoId(dto.UserInfoId)
			for _, role := range roles {
				ok, _ := casbinEnforcer().Enforce(role, c.Request.RequestURI, c.Request.Method)
				if ok {
					c.Next()
					return
				}
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
