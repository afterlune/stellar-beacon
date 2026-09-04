package middlewares

import (
	"benetnasch/app/application/service"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/shared"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/casbin/casbin/v2"
	model2 "github.com/casbin/casbin/v2/model"
	casbinutil "github.com/casbin/casbin/v2/util"
	xormadapter "github.com/casbin/xorm-adapter/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type bodyLog struct {
	gin.ResponseWriter
	body         *bytes.Buffer
	captureLimit int
}

type requestBodyCapture struct {
	data    []byte
	omitted bool
}

// replayReadCloser restores the bytes consumed for bounded request logging
// and closes the original body when the handler/server is done with it.
type replayReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r replayReadCloser) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

const (
	operationLogRequestParamLimit = 2000
	operationLogResponseLimit     = 10000
	requestBodyCaptureLimit       = 64 * 1024
)

func captureRequestBody(req *http.Request) (requestBodyCapture, error) {
	if req == nil || req.Body == nil {
		return requestBodyCapture{}, nil
	}
	if req.ContentLength == 0 || req.ContentLength > requestBodyCaptureLimit || isBinaryContentType(req.Header.Get("Content-Type")) {
		return requestBodyCapture{omitted: req.ContentLength != 0}, nil
	}

	original := req.Body
	data, err := io.ReadAll(io.LimitReader(original, requestBodyCaptureLimit+1))
	// The bounded prefix must be replayed before the unread remainder, so
	// handlers receive the same body even when it exceeds the log capture cap.
	remainder := io.MultiReader(bytes.NewReader(data), original)
	req.Body = replayReadCloser{Reader: remainder, closer: original}
	if err != nil {
		return requestBodyCapture{omitted: true}, err
	}
	if len(data) > requestBodyCaptureLimit {
		return requestBodyCapture{omitted: true}, nil
	}
	return requestBodyCapture{data: data}, nil
}

func requestBodyLogPayload(req *http.Request, capture requestBodyCapture) string {
	if capture.omitted {
		return "[request body omitted: binary or larger than capture limit]"
	}
	return requestLogPayload(req, capture.data)
}

func requestLogPayload(req *http.Request, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	contentType := ""
	if req != nil {
		contentType = req.Header.Get("Content-Type")
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType == "" {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	if mediaType == "" {
		mediaType = "unknown"
	}

	if isBinaryContentType(mediaType) || !utf8.Valid(body) {
		return fmt.Sprintf("[request body omitted: content-type=%s, bytes=%d]", mediaType, len(body))
	}
	return truncateLogText(string(body), operationLogRequestParamLimit)
}

func isBinaryContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.HasPrefix(contentType, "multipart/") ||
		strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "audio/") ||
		strings.HasPrefix(contentType, "video/") ||
		contentType == "application/octet-stream" ||
		contentType == "application/pdf" ||
		contentType == "application/zip"
}

func truncateLogText(value string, limit int) string {
	if !utf8.ValidString(value) {
		return "[text omitted: invalid UTF-8]"
	}
	if len(value) <= limit {
		return value
	}

	const suffix = "...[truncated]"
	cut := limit - len(suffix)
	for cut > 0 && !utf8.ValidString(value[:cut]) {
		cut--
	}
	return value[:cut] + suffix
}

func responseCodeFromBody(body []byte) (int, bool) {
	if len(bytes.TrimSpace(body)) == 0 {
		return 0, false
	}
	var envelope struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return 0, false
	}
	return envelope.Code, true
}

func (w bodyLog) Write(b []byte) (int, error) {
	if w.body != nil && w.captureLimit > w.body.Len() {
		remaining := w.captureLimit - w.body.Len()
		if len(b) > remaining {
			_, _ = w.body.Write(b[:remaining])
		} else {
			_, _ = w.body.Write(b)
		}
	}
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
		slog.Error("failed to open swagger.json", "error_code", apperrors.SafeCode(err))
		return
	}
	defer open.Close()

	bys, err := io.ReadAll(open)
	if err != nil {
		slog.Error("failed to read swagger.json", "error_code", apperrors.SafeCode(err))
		return
	}

	err = json.Unmarshal(bys, &swaggerCache)
	if err != nil {
		slog.Error("failed to unmarshal swagger.json", "error_code", apperrors.SafeCode(err))
		return
	}
	slog.Info("swagger cache initialized successfully")
}

// getSwaggerInfo 从缓存中获取 swagger 信息
func getSwaggerInfo(reqURI, reqMethod string) (module, desc string) {
	// 使用缓存的 swagger 内容
	hm := swaggerCache
	if hm == nil {
		slog.Error("swagger cache not initialized")
		return "Unknown", "Unknown"
	}

	apis, ok := hm["paths"].(map[string]interface{})
	if !ok {
		slog.Error("invalid swagger format: paths not found")
		return "Unknown", "Unknown"
	}

	pathData, ok := findSwaggerPath(apis, reqURI)
	if !ok {
		// A route can be valid without having generated Swagger metadata (for
		// example an optional integration-only endpoint).  This is not a
		// malformed request and must not pollute the error log.
		slog.Debug("swagger metadata not found")
		return "Unknown", "Unknown"
	}

	reqMethodData, ok := pathData[strings.ToLower(reqMethod)].(map[string]interface{})
	if !ok {
		slog.Debug("swagger method metadata not found")
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

func findSwaggerPath(apis map[string]interface{}, reqURI string) (map[string]interface{}, bool) {
	if pathData, ok := apis[reqURI].(map[string]interface{}); ok {
		return pathData, true
	}

	requestSegments := splitPathSegments(reqURI)
	bestScore := -1
	var best map[string]interface{}
	for pattern, rawPathData := range apis {
		pathData, ok := rawPathData.(map[string]interface{})
		if !ok {
			continue
		}
		patternSegments := splitPathSegments(pattern)
		if len(patternSegments) != len(requestSegments) {
			continue
		}

		score := 0
		matched := true
		for index, patternSegment := range patternSegments {
			if isSwaggerPathParameter(patternSegment) {
				continue
			}
			if patternSegment != requestSegments[index] {
				matched = false
				break
			}
			score++
		}
		if matched && score > bestScore {
			bestScore = score
			best = pathData
		}
	}
	return best, bestScore >= 0
}

func splitPathSegments(value string) []string {
	trimmed := strings.Trim(value, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func isSwaggerPathParameter(segment string) bool {
	return strings.HasPrefix(segment, ":") ||
		(strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}"))
}

func Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqURI := strings.Split(c.Request.RequestURI, "?")[0]
		// Public Agent prompts and streamed answers may contain visitor content;
		// they are intentionally excluded from the generic operation-log sink.
		// Agent prompts and time-capsule bodies may contain private user
		// content. Keep them out of the generic operation-log payload while
		// still allowing the request to flow through the normal middleware.
		captureResponse := reqURI != "/agent/chat" && !isTimeCapsulePath(reqURI) && !isSpaceCompanionPath(reqURI)
		blw := &bodyLog{body: bytes.NewBufferString(""), captureLimit: operationLogResponseLimit, ResponseWriter: c.Writer}
		if !captureResponse {
			blw.captureLimit = 0
		}
		c.Writer = blw
		capture := requestBodyCapture{}
		var err error
		if captureResponse {
			capture, err = captureRequestBody(c.Request)
		}
		if err != nil {
			slog.Error("read request body failed", "error_code", apperrors.SafeCode(err))
		}

		c.Next()
		if (c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete) && captureResponse {
			reqMethod := c.Request.Method
			ip := shared.GetIpAddress(c.Request)
			ipSource := shared.GetIpSource(ip)

			dto, authenticated := userDetailsFromContext(c)
			if !authenticated {
				dto = port.UserDetailsDTO{
					Nickname:   "访客",
					UserInfoId: -1,
				}
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
			requestParam := requestBodyLogPayload(c.Request, capture)
			resData := truncateLogText(blw.body.String(), operationLogResponseLimit)
			optLog := port.TOperationLog{
				OptModule:     module,
				OptType:       optType,
				OptUri:        reqURI,
				OptMethod:     optFunc,
				OptDesc:       desc,
				RequestParam:  requestParam,
				RequestMethod: reqMethod,
				ResponseData:  resData,
				UserId:        userId,
				Nickname:      nickname,
				IpAddress:     ip,
				IpSource:      ipSource,
			}
			repository.EnqueueOptLog(optLog)
		}
		responseCode, responseDecoded := responseCodeFromBody(blw.body.Bytes())
		if captureResponse && responseDecoded && responseCode == 51000 {
			reqMethod := c.Request.Method
			ip := shared.GetIpAddress(c.Request)
			ipSource := shared.GetIpSource(ip)
			optFunc := c.HandlerName()

			// 获取 swagger 信息
			_, desc := getSwaggerInfo(reqURI, reqMethod)

			exLog := port.TExceptionLog{
				OptUri:        reqURI,
				OptMethod:     optFunc,
				RequestMethod: reqMethod,
				RequestParam:  requestBodyLogPayload(c.Request, capture),
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
	var userVO port.UserVO
	err := c.ShouldBind(&userVO)
	if err != nil {
		slog.Error("bind login request failed", "error_code", apperrors.SafeCode(err))
		return model.ResultFailWithMessage("登录失败，请联系管理员")
	}
	if !shared.CheckEmail(userVO.Username) {
		return model.ResultFailWithMessage("邮箱格式不正确！")
	}
	ctx := c.Request.Context()
	ipAddress := shared.GetIpAddress(c.Request)
	allowed, err := loginAttemptAllowed(ctx, userVO.Username, ipAddress)
	if err != nil {
		slog.Error("login rate limiter failed", "error_code", apperrors.SafeCode(err))
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if !allowed {
		return model.ResultFailWithMessage("账号或密码不正确！")
	}
	userDetailsDTO, err := userAuthService.Authenticate(ctx, userVO)
	if err != nil {
		slog.Error("authenticate user failed", "error_code", apperrors.SafeCode(err))
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if userDetailsDTO == nil {
		if err := recordLoginFailure(ctx, userVO.Username, ipAddress); err != nil {
			slog.Error("record login failure", "error_code", apperrors.SafeCode(err))
		}
		return model.ResultFailWithMessage("账号或密码不正确！")
	}
	if err := clearLoginFailures(ctx, userVO.Username, ipAddress); err != nil {
		slog.Error("clear login failures", "error_code", apperrors.SafeCode(err))
	}
	region := shared.GetIpSource(ipAddress)
	userAuth := port.TUserAuth{
		Id:            userDetailsDTO.Id,
		IpAddress:     ipAddress,
		IpSource:      region,
		LastLoginTime: time.Now(),
	}
	if err := userAuthService.UpdateUserIp(ctx, userAuth); err != nil {
		slog.Error("update login metadata failed", "error_code", apperrors.SafeCode(err))
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}

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
		slog.Error("failed to create token", "error_code", apperrors.SafeCode(err))
		return model.ResultFailWithMessage("登录失败")
	}
	userInfoDTO := projectUserInfoDTO(*userDetailsDTO, accessToken)

	return model.ResultOkWithData(userInfoDTO)
}

// projectUserInfoDTO explicitly selects the fields safe for the login
// response. Keeping this projection typed avoids a JSON round trip and makes
// it impossible for password, roles, or future private fields to leak by
// accident.
func projectUserInfoDTO(details port.UserDetailsDTO, token string) model.UserInfoDTO {
	return model.UserInfoDTO{
		Id:            details.Id,
		UserInfoId:    details.UserInfoId,
		Email:         details.Email,
		LoginType:     details.LoginType,
		Username:      details.Username,
		Nickname:      details.Nickname,
		Avatar:        details.Avatar,
		Intro:         details.Intro,
		Website:       details.Website,
		IpAddress:     details.IpAddress,
		IpSource:      details.IpSource,
		IsSubscribe:   details.IsSubscribe,
		LastLoginTime: details.LastLoginTime,
		Token:         token,
	}
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
		// 后台接口、需要用户身份的个人资料接口、胶囊接口和登出接口需要认证；
		// 注册、找回密码和公开用户资料接口保持公开。
		if !isAdminPath(c.Request.URL.Path) && !isProtectedUserPath(c.Request.URL.Path, c.Request.Method) && c.Request.URL.Path != "/users/logout" && !isTimeCapsulePath(c.Request.URL.Path) {
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
				slog.Error("token validation failed", "error_code", apperrors.SafeCode(err))
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
			return
		}
		userAuthID, ok := hm["sub"].(string)
		if !ok || userAuthID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
			return
		}
		var userDetailsDTO port.UserDetailsDTO
		dto, err := shared.HGetCtx(c.Request.Context(), shared.LOGIN_USER, userAuthID)
		if err != nil {
			slog.Error("load login user failed", "error_code", apperrors.SafeCode(err))
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, model.ResultFailWithMessage("服务暂不可用"))
			return
		}
		if dto == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithCode(41000))
			return
		}
		if err = json.Unmarshal([]byte(dto), &userDetailsDTO); err != nil {
			slog.Error("decode cached login user failed", "error_code", apperrors.SafeCode(err))
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

func isTimeCapsulePath(path string) bool {
	return path == "/capsules" || strings.HasPrefix(path, "/capsules/")
}

func isSpaceCompanionPath(path string) bool {
	return path == "/internal/space" || strings.HasPrefix(path, "/internal/space/")
}

func isProtectedUserPath(path, method string) bool {
	switch {
	case path == "/users/info" && method == http.MethodPut:
		return true
	case path == "/users/avatar" && method == http.MethodPost:
		return true
	case path == "/users/email" && method == http.MethodPut:
		return true
	case path == "/users/subscribe" && method == http.MethodPut:
		return true
	default:
		return false
	}
}

// userDetailsFromContext is the single boundary for middleware identity
// values. Middleware order and test doubles must not be able to turn a bad
// context value into a process-level panic.
func userDetailsFromContext(c *gin.Context) (port.UserDetailsDTO, bool) {
	if c == nil {
		return port.UserDetailsDTO{}, false
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return port.UserDetailsDTO{}, false
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok || dto.UserInfoId <= 0 {
		return port.UserDetailsDTO{}, false
	}
	return dto, true
}

// getRolesByUserInfoId 获取用户角色，优先从缓存中获取。
// Cache failures are soft and fall back to the repository; repository and
// configuration failures are returned so authorization can distinguish an
// unavailable permission system from a user with no roles.
func getRolesByUserInfoId(ctx context.Context, userInfoId int) ([]string, error) {
	cacheKey := fmt.Sprintf("roles:%d", userInfoId)
	if roleCache != nil {
		cachedRoles, err := roleCache.HGet(ctx, "user_roles", cacheKey)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(ctx, "load cached roles failed", "error_code", apperrors.SafeCode(err))
		}
		if cachedRoles != "" {
			var roles []string
			if err := json.Unmarshal([]byte(cachedRoles), &roles); err == nil {
				return roles, nil
			}
			slog.WarnContext(ctx, "discard invalid cached roles", "reason", "invalid_json")
		}
	}

	if roleRepo == nil {
		return nil, errors.New("role repository is not configured")
	}
	roles, err := roleRepo.ListRolesByUserInfoID(ctx, userInfoId)
	if err != nil {
		return nil, err
	}
	if roleCache != nil {
		rolesJSON, err := json.Marshal(roles)
		if err != nil {
			return nil, err
		}
		if err := roleCache.HSet(ctx, "user_roles", cacheKey, rolesJSON, 1*time.Hour); err != nil {
			slog.WarnContext(ctx, "cache user roles failed", "error_code", apperrors.SafeCode(err))
		}
	}
	return roles, nil
}

// checkPermission checks the legacy Casbin policy first and then falls back to
// the resource/role tables used by the admin UI. Older installations keep
// their policies in casbin_rule, while newer migrations seed t_resource and
// t_role_resource. Checking both keeps existing deployments compatible and
// makes newly configured resources effective at runtime.
func checkPermission(ctx context.Context, roles []string, uri, method string) (bool, string, error) {
	enforcer, err := casbinEnforcer()
	if err != nil {
		return false, "", err
	}
	for _, role := range roles {
		ok, err := enforcer.Enforce(role, uri, method)
		if err != nil {
			slog.ErrorContext(ctx, "permission check failed", "error_code", apperrors.SafeCode(err))
			return false, "", err
		}
		if ok {
			return true, "", nil
		}
	}

	if roleRepo != nil {
		resources, err := roleRepo.ListResourceRoles(ctx)
		if err != nil {
			return false, "", err
		}
		if resourcePermissionGranted(roles, resources, uri, method) {
			return true, "", nil
		}
	}

	return false, fmt.Sprintf("用户角色 %v 没有权限访问 %s %s", roles, method, uri), nil
}

// resourcePermissionGranted is deliberately kept pure so the authorization
// semantics can be tested without a database or a Casbin singleton. Resource
// URLs use Casbin's segment-aware KeyMatch2 semantics. This preserves the
// existing trailing `/*` resource format while preventing a resource such as
// `/reviews/*/approve` from accidentally authorizing `/reviews/42/reject`.
func resourcePermissionGranted(roles []string, resources []port.ResourceRoleView, uri, method string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	uri = strings.TrimSpace(uri)
	if uri == "" || method == "" {
		return false
	}
	for _, resource := range resources {
		if !strings.EqualFold(strings.TrimSpace(resource.RequestMethod), method) ||
			!casbinutil.KeyMatch2(uri, strings.TrimSpace(resource.Url)) {
			continue
		}
		for _, resourceRole := range resource.RoleList {
			resourceRole = strings.TrimSpace(resourceRole)
			for _, role := range roles {
				if resourceRole != "" && resourceRole == strings.TrimSpace(role) {
					return true
				}
			}
		}
	}
	return false
}

func CasbinResourceFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAdminPath(c.Request.URL.Path) {
			dto, ok := userDetailsFromContext(c)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithStatus(model.NO_LOGIN))
				return
			}
			roles, err := getRolesByUserInfoId(c.Request.Context(), dto.UserInfoId)
			if err != nil {
				slog.ErrorContext(c.Request.Context(), "load roles for authorization failed", "error_code", apperrors.SafeCode(err))
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, model.ResultFailWithMessage("权限服务暂不可用"))
				return
			}
			ok, deniedMessage, err := checkPermission(c.Request.Context(), roles, c.Request.URL.Path, c.Request.Method)
			if err != nil {
				slog.Error("authorization check failed", "error_code", apperrors.SafeCode(err))
				c.AbortWithStatusJSON(http.StatusInternalServerError, model.ResultFailWithMessage("权限检查失败"))
				return
			}
			if deniedMessage != "" {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage(deniedMessage))
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
	roleRepo        port.RoleRepository
	roleCache       port.Cache
	userAuthService service.UserAuthService = new(service.MyUserAuthService)
)

func ConfigureRoleRepository(repo port.RoleRepository) {
	roleRepo = repo
}

// ConfigureRoleCache injects the application-facing cache into middleware.
// It keeps role lookups testable and prevents this package from silently
// constructing a second global Redis client.
func ConfigureRoleCache(cache port.Cache) {
	roleCache = cache
}

// ConfigureUserAuthService injects the application authentication service
// used by the login filter. Keeping this wiring in the composition root avoids
// falling back to an unconfigured service instance.
func ConfigureUserAuthService(auth service.UserAuthService) {
	if auth != nil {
		userAuthService = auth
	}
}

func AccessLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
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
				slog.Info("cleaned up old IP limiters", "count", count)
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
	adapterErr   error
	enforcerErr  error
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
func casbinAdapter() (*xormadapter.Adapter, error) {
	adapterOnce.Do(func() {
		engine := ormInit.GetEngine()
		if engine == nil {
			adapterErr = fmt.Errorf("database engine is not initialized")
			return
		}
		a, err := xormadapter.NewAdapterByEngine(engine)
		if err != nil {
			adapterErr = err
			return
		}
		xormAdapter = a
	})
	return xormAdapter, adapterErr
}

func casbinEnforcer() (*casbin.Enforcer, error) {
	enforcerOnce.Do(func() {
		m, err := model2.NewModelFromString(text)
		if err != nil {
			enforcerErr = err
			return
		}

		adapter, err := casbinAdapter()
		if err != nil {
			enforcerErr = err
			return
		}
		e, err := casbin.NewEnforcer(m, adapter)
		if err != nil {
			enforcerErr = err
			return
		}

		err = e.LoadPolicy()
		if err != nil {
			enforcerErr = err
			return
		}
		enforcer = e
	})

	return enforcer, enforcerErr
}
