package middlewares

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/repository"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/shared"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/visitor"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type bodyLog struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

const (
	operationLogRequestParamLimit = 2000
	operationLogResponseLimit     = 10000
)

func isRecommendationQuery(method, path string) bool {
	return method == http.MethodPost && path == "/v1/auth/me/recommendations/query"
}

func shouldRecordOperation(method, path string) bool {
	if isRecommendationQuery(method, path) {
		return false
	}
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete
}

func requestLogPayload(req *http.Request, body []byte) string {
	if req != nil && isRecommendationQuery(req.Method, req.URL.Path) {
		return "[request body omitted: recommendation seeds are request-only]"
	}
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

func (w bodyLog) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// 缓存 swagger.json 内容
var swaggerCache map[string]interface{}

// 配置项
const (
	SwaggerFilePath = "docs/api/openapi.json"
)

func init() {
	// 初始化 swagger 缓存
	swaggerCache = make(map[string]interface{})
	open, err := os.Open(SwaggerFilePath)
	if err != nil {
		slog.Error("failed to open swagger.json", "error", err)
		return
	}
	defer open.Close()

	bys, err := io.ReadAll(open)
	if err != nil {
		slog.Error("failed to read swagger.json", "error", err)
		return
	}

	err = json.Unmarshal(bys, &swaggerCache)
	if err != nil {
		slog.Error("failed to unmarshal swagger.json", "error", err)
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

	pathData, ok := swaggerPathData(apis, reqURI)
	if !ok {
		slog.Error("invalid swagger format: path not found")
		return "Unknown", "Unknown"
	}

	reqMethodData, ok := pathData[strings.ToLower(reqMethod)].(map[string]interface{})
	if !ok {
		slog.Error("invalid swagger format: method not found")
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

func swaggerPathData(apis map[string]interface{}, reqURI string) (map[string]interface{}, bool) {
	if value, ok := apis[reqURI].(map[string]interface{}); ok {
		return value, true
	}
	// Static segments must win over parameterised ones: /comments/batch and
	// /comments/{commentId} both match a batch request, and map iteration order
	// would otherwise decide which metadata is used.
	type candidate struct {
		template string
		data     map[string]interface{}
		params   int
	}
	var matches []candidate
	for template, value := range apis {
		pathData, ok := value.(map[string]interface{})
		if !ok || !swaggerPathMatches(template, reqURI) {
			continue
		}
		matches = append(matches, candidate{template: template, data: pathData, params: swaggerPathParamCount(template)})
	}
	if len(matches) == 0 {
		return nil, false
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].params != matches[j].params {
			return matches[i].params < matches[j].params
		}
		return matches[i].template < matches[j].template
	})
	return matches[0].data, true
}

func swaggerPathParamCount(template string) int {
	count := 0
	for _, part := range splitSwaggerPath(template) {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			count++
		}
	}
	return count
}

func swaggerPathMatches(template, request string) bool {
	templateParts := splitSwaggerPath(template)
	requestParts := splitSwaggerPath(request)
	if len(templateParts) != len(requestParts) {
		return false
	}
	for index, templatePart := range templateParts {
		if strings.HasPrefix(templatePart, "{") && strings.HasSuffix(templatePart, "}") {
			if requestParts[index] == "" {
				return false
			}
			continue
		}
		if templatePart != requestParts[index] {
			return false
		}
	}
	return true
}

func splitSwaggerPath(value string) []string {
	value = strings.Trim(value, "/")
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

// shouldRecordException reports whether a failed response deserves an
// exception-log row. A canceled request context means the caller is gone: the
// handler failed because the connection went away, not because the service did,
// so recording it would only create false positives. Operation logs are still
// written, because an aborted write remains worth auditing.
func shouldRecordException(ctx context.Context, code string) bool {
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	return code == "OPERATION_FAILED" || code == "INVALID_ARGUMENT"
}

func Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/readyz" {
			c.Next()
			return
		}
		blw := &bodyLog{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		// 创建一个缓冲区
		var buf bytes.Buffer

		// 拷贝数据到缓冲区
		_, err := io.Copy(&buf, c.Request.Body)
		if err != nil {
			slog.Error("read request body failed", "error", err)
		}

		// 获取缓冲区中的数据
		reqData := buf.Bytes()

		// 创建一个新的io.ReadCloser
		c.Request.Body = io.NopCloser(bytes.NewReader(reqData))

		c.Next()
		if shouldRecordOperation(c.Request.Method, c.Request.URL.Path) {
			reqURI := strings.Split(c.Request.RequestURI, "?")[0]
			reqMethod := c.Request.Method
			ip := visitor.ClientIP(c.Request.Context(), c.Request)
			ipSource := visitor.RegionForIP(c.Request.Context(), ip)

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
			requestParam := requestLogPayload(c.Request, reqData)
			resData := truncateLogText(blw.body.String(), operationLogResponseLimit)
			optLog := entity.TOperationLog{
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
		resData := make(map[string]interface{})
		json.Unmarshal(blw.body.Bytes(), &resData)
		code, _ := resData["code"].(string)
		if shouldRecordException(c.Request.Context(), code) {
			reqURI := strings.Split(c.Request.RequestURI, "?")[0]
			reqMethod := c.Request.Method
			ip := visitor.ClientIP(c.Request.Context(), c.Request)
			ipSource := visitor.RegionForIP(c.Request.Context(), ip)
			optFunc := c.HandlerName()

			// 获取 swagger 信息
			_, desc := getSwaggerInfo(reqURI, reqMethod)

			exLog := entity.TExceptionLog{
				OptUri:        reqURI,
				OptMethod:     optFunc,
				RequestMethod: reqMethod,
				RequestParam:  requestLogPayload(c.Request, reqData),
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
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/readyz" || isSeoRoute(c.Request.URL.Path) {
			c.Next()
			return
		}
		if visitor.IsBot(c.Request) {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ResultFailWithMessage("You may be a robot！"))
			return
		}
		c.Next()
	}
}

func isSeoRoute(path string) bool {
	return path == "/sitemap.xml" || path == "/robots.txt" || path == "/feed.xml" ||
		strings.HasPrefix(path, "/articles/") || strings.HasPrefix(path, "/u/")
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
		if uri == "/v1/auth/login" && req.Method == http.MethodPost {
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
		slog.Error("bind login request failed", "error", err)
		return model.ResultFailWithMessage("登录失败，请联系管理员")
	}
	if !shared.CheckEmail(userVO.Username) {
		return model.ResultFailWithMessage("邮箱格式不正确！")
	}
	ctx := c.Request.Context()
	ipAddress := visitor.ClientIP(ctx, c.Request)
	allowed, err := loginAttemptAllowed(ctx, userVO.Username, ipAddress)
	if err != nil {
		slog.Error("login rate limiter failed", "error", err)
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if !allowed {
		return model.ResultFailWithMessage("账号或密码不正确！")
	}
	userDetailsDTO, err := userAuthService.Authenticate(ctx, userVO)
	if err != nil {
		slog.Error("authenticate user failed", "error", err)
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}
	if userDetailsDTO == nil {
		if err := recordLoginFailure(ctx, userVO.Username, ipAddress); err != nil {
			slog.Error("record login failure", "error", err)
		}
		return model.ResultFailWithMessage("账号或密码不正确！")
	}
	if err := clearLoginFailures(ctx, userVO.Username, ipAddress); err != nil {
		slog.Error("clear login failures", "error", err)
	}
	region := visitor.RegionForIP(ctx, ipAddress)
	userAuth := entity.TUserAuth{
		Id:            userDetailsDTO.Id,
		IpAddress:     ipAddress,
		IpSource:      region,
		LastLoginTime: time.Now(),
	}
	if err := userAuthService.UpdateUserIp(ctx, userAuth); err != nil {
		slog.Error("update login metadata failed", "error", err)
		return model.ResultFailWithMessage("系统繁忙，请稍后再试")
	}

	regionlist := strings.Split(region, "|")
	ipSource := "Unknown"
	if len(regionlist) >= 4 {
		ipSource = regionlist[2] + "|" + regionlist[3]
	}
	userDetailsDTO.IpAddress = ipAddress
	userDetailsDTO.IpSource = ipSource
	name, version := visitor.Browser(c.Request)
	osName := visitor.OS(c.Request)
	userDetailsDTO.Browser = name + version
	userDetailsDTO.Os = osName

	accessToken, _, err := shared.CreateTokenCtx(ctx, userDetailsDTO)
	if err != nil {
		slog.Error("failed to create token", "error", err)
		return model.ResultFailWithMessage("登录失败")
	}
	var userInfoDTO model.UserInfoDTO
	marshal, err := json.Marshal(userDetailsDTO)
	if err != nil {
		slog.Error("marshal login user failed", "error", err)
	}

	err = json.Unmarshal(marshal, &userInfoDTO)
	if err != nil {
		slog.Error("decode login user failed", "error", err)
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
		// 后台接口、当前用户接口和登出接口需要认证；其余前台接口保持公开，
		// 但公开接口仍需要识别已登录读者（评论、私密文章访问等）。
		if !requiresAuthentication(c.Request.URL.Path) {
			attachOptionalLoginUser(c)
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
				slog.Error("token validation failed", "error", err)
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
			slog.Error("load login user failed", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, model.ResultFailWithMessage("服务暂不可用"))
			return
		}
		if dto == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithCode(41000))
			return
		}
		if err = json.Unmarshal([]byte(dto), &userDetailsDTO); err != nil {
			slog.Error("decode cached login user failed", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, model.ResultFailWithMessage("server error"))
			return
		}
		if userDetailsDTO.UserInfoId <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithCode(41000))
			return
		}
		enabled, err := isUserEnabled(c.Request.Context(), userDetailsDTO.UserInfoId)
		if err != nil {
			slog.Error("check authenticated user status failed", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, model.ResultFailWithMessage("服务暂不可用"))
			return
		}
		if !enabled {
			revokeCachedUserSession(c.Request.Context(), userAuthID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithCode(41000))
			return
		}
		userDetailsDTO.LastLoginTime = time.Now()
		c.Set("userInfo", userDetailsDTO)
		c.Next()
	}
}

// attachOptionalLoginUser sets userInfo for a public route when the request
// carries a usable bearer token. Public routes must stay reachable with a
// missing, stale or malformed token: those requests simply stay anonymous.
func attachOptionalLoginUser(c *gin.Context) {
	authorization := strings.TrimSpace(c.Request.Header.Get(shared.TOKEN_HEADER))
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "null" {
		return
	}
	hm, err := shared.TokenParseCtx(c.Request.Context(), parts[1])
	if err != nil || hm == nil {
		return
	}
	userAuthID, ok := hm["sub"].(string)
	if !ok || userAuthID == "" {
		return
	}
	dto, err := shared.HGetCtx(c.Request.Context(), shared.LOGIN_USER, userAuthID)
	if err != nil || dto == "" {
		return
	}
	var userDetailsDTO model.UserDetailsDTO
	if err := json.Unmarshal([]byte(dto), &userDetailsDTO); err != nil {
		return
	}
	if userDetailsDTO.UserInfoId <= 0 {
		return
	}
	enabled, err := isUserEnabled(c.Request.Context(), userDetailsDTO.UserInfoId)
	if err != nil {
		slog.Warn("check optional login user status failed", "error", err)
		return
	}
	if !enabled {
		revokeCachedUserSession(c.Request.Context(), userAuthID)
		return
	}
	c.Set("userInfo", userDetailsDTO)
}

func isUserEnabled(ctx context.Context, userInfoID int) (bool, error) {
	if userInfoRepo == nil {
		return false, fmt.Errorf("user info repository is not configured")
	}
	enabled, err := userInfoRepo.IsEnabled(ctx, userInfoID)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func revokeCachedUserSession(ctx context.Context, userAuthID string) {
	if err := shared.HDelCtx(ctx, shared.LOGIN_USER, userAuthID); err != nil {
		slog.Warn("remove disabled user session failed", "error", err)
	}
	if err := shared.DelCtx(ctx, shared.REFRESH_TOKEN_PREFIX+userAuthID); err != nil {
		slog.Warn("remove disabled user refresh token failed", "error", err)
	}
}

func isAdminPath(path string) bool {
	return path == "/v1/admin" || strings.HasPrefix(path, "/v1/admin/")
}

func requiresAuthentication(path string) bool {
	return isAdminPath(path) ||
		strings.HasPrefix(path, "/v1/studio") ||
		path == "/v1/auth/logout" ||
		path == "/v1/auth/me" ||
		strings.HasPrefix(path, "/v1/auth/me/")
}

// permissionPath maps the versioned resource names back to the permission
// records stored in the existing database. This is an authorization adapter,
// not an HTTP compatibility route: legacy URLs are still not registered.
func permissionPath(path string) string {
	path = strings.TrimPrefix(path, "/v1")
	path = strings.Replace(path, "/admin/dashboard", "/admin", 1)
	path = strings.Replace(path, "/admin/site", "/admin/website/config", 1)
	path = strings.Replace(path, "/admin/friend-links", "/admin/links", 1)
	path = strings.Replace(path, "/admin/albums/cover", "/admin/photos/albums/upload", 1)
	path = strings.Replace(path, "/admin/albums/options", "/admin/photos/albums/info", 1)
	path = strings.Replace(path, "/admin/albums", "/admin/photos/albums", 1)
	path = strings.Replace(path, "/admin/permissions", "/admin/resources", 1)
	path = strings.Replace(path, "/admin/roles/resource-options", "/admin/role/resources", 1)
	path = strings.Replace(path, "/admin/roles/menu-options", "/admin/role/menus", 1)
	path = strings.Replace(path, "/admin/users/roles", "/admin/users/role", 1)
	path = strings.Replace(path, "/admin/users/areas", "/admin/users/area", 1)
	path = strings.Replace(path, "/admin/me/menu", "/admin/user/menus", 1)
	path = strings.Replace(path, "/admin/menus/", "/admin/menus/isHidden/", 1)
	path = strings.Replace(path, "/admin/articles/featured", "/admin/articles/topAndFeatured", 1)
	path = strings.Replace(path, "/admin/articles/trash", "/admin/articles", 1)
	path = strings.Replace(path, "/admin/articles/batch-delete", "/admin/articles/delete", 1)
	path = strings.Replace(path, "/admin/photos/trash", "/admin/photos/delete", 1)
	path = strings.Replace(path, "/admin/jobs/groups", "/admin/jobs/jobGroups", 1)
	path = strings.Replace(path, "/admin/logs/jobs/clean", "/admin/jobLogs/clean", 1)
	path = strings.Replace(path, "/admin/logs/jobs/groups", "/admin/jobLogs/jobGroups", 1)
	path = strings.Replace(path, "/admin/logs/jobs", "/admin/jobLogs", 1)
	path = strings.Replace(path, "/admin/logs/operations", "/admin/operation/logs", 1)
	path = strings.Replace(path, "/admin/logs/exceptions", "/admin/exception/logs", 1)
	return path
}

func checkUserResourcePermission(ctx context.Context, userInfoID int, path, method string) (bool, error) {
	if roleRepo == nil {
		return false, fmt.Errorf("role repository is not configured")
	}
	if userInfoID <= 0 {
		return false, nil
	}
	return roleRepo.HasUserResourcePermission(ctx, userInfoID, permissionPath(path), method)
}

func ResourceAuthorizationFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAdminPath(c.Request.URL.Path) {
			value, ok := c.Get("userInfo")
			if !ok {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage("权限不足"))
				return
			}
			dto, ok := value.(model.UserDetailsDTO)
			if !ok {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage("权限不足"))
				return
			}
			permissionResource, permissionMethod := permissionTarget(c.Request.URL.Path, c.Request.Method)
			allowed, err := checkUserResourcePermission(c.Request.Context(), dto.UserInfoId, permissionResource, permissionMethod)
			if err != nil {
				slog.Error("authorization check failed", "error", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, model.ResultFailWithMessage("权限检查失败"))
				return
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusOK, model.ResultFailWithMessage("权限不足"))
				return
			}
			if permissionResource == c.Request.URL.Path && permissionMethod == c.Request.Method {
				c.Set("adminResourceAuthorized", true)
			}
		}
		c.Next()
	}
}

// Incident handling notes share the same audience as the health monitor. This
// keeps existing read-only monitor grants consistent without granting general
// admin POST access.
func permissionTarget(path, method string) (string, string) {
	if method == http.MethodPost && strings.HasPrefix(path, "/v1/admin/monitor/incidents/") && strings.HasSuffix(path, "/updates") {
		return "/v1/admin/monitor/health", http.MethodGet
	}
	return path, method
}

// 配置项
const (
	IpLimiterMaxSize = 10000 // ipLimiter map 的最大容量
)

type ipRateLimiters struct {
	read  *rate.Limiter
	write *rate.Limiter
}

func newIPRateLimiters() *ipRateLimiters {
	return &ipRateLimiters{
		read:  rate.NewLimiter(rate.Every(time.Minute/240), 240),
		write: rate.NewLimiter(rate.Every(time.Minute/60), 60),
	}
}

// isReadLimitedRequest keeps body-based recommendation queries on the read
// limiter. The endpoint is a read-only POST because local reading seeds must
// never be placed in a URL or access log.
func isReadLimitedRequest(method, path string) bool {
	if method == http.MethodGet {
		// The verification-code endpoint sends mail even though it is a GET, so
		// it stays on the write budget. Every other GET is a read surface.
		return path != "/v1/auth/verification-code"
	}
	return isRecommendationQuery(method, path)
}

var (
	globalLimiter   = rate.NewLimiter(rate.Every(time.Second/20), 1200)
	ipLimiter       = make(map[string]*ipRateLimiters)
	mutex           sync.Mutex
	roleRepo        port.RoleRepository
	userInfoRepo    port.UserInfoRepository
	userAuthService service.UserAuthService = new(service.MyUserAuthService)
)

func ConfigureRoleRepository(repo port.RoleRepository) {
	roleRepo = repo
}

func ConfigureUserInfoRepository(repo port.UserInfoRepository) {
	userInfoRepo = repo
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
		if authorized, ok := c.Get("adminResourceAuthorized"); ok {
			if allowed, isBool := authorized.(bool); isBool && allowed {
				c.Next()
				return
			}
		}

		ip := visitor.ClientIP(c.Request.Context(), c.Request)
		mutex.Lock()
		limiters, ok := ipLimiter[ip]
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
			limiters = newIPRateLimiters()
			ipLimiter[ip] = limiters
		}
		mutex.Unlock()

		limiter := limiters.write
		if isReadLimitedRequest(c.Request.Method, c.Request.URL.Path) {
			limiter = limiters.read
		}
		if !limiter.Allow() || !globalLimiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.ResultFailWithMessage("请求过于频繁"))
			return
		}
		c.Next()
	}
}
