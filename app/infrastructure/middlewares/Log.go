package middlewares

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"benetnasch/app/infrastructure/zlog"
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"os"
	"strings"
)

type bodyLog struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLog) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
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
			nickname := shared.GetUserDetailsDTO().Nickname
			userId := shared.GetUserDetailsDTO().UserInfoId
			optFunc := c.HandlerName()
			open, err := os.Open("docs/swagger.json")
			defer open.Close()
			if err != nil {
				zlog.Error(err.Error())
			}
			bys, _ := io.ReadAll(open)
			hm := make(map[string]interface{})
			err = json.Unmarshal(bys, &hm)
			if err != nil {
				zlog.Error(err.Error())
			}
			apis := hm["paths"].(map[string]interface{})
			data := apis[reqURI].(map[string]interface{})[strings.ToLower(reqMethod)].(map[string]interface{})
			module := data["summary"].(string)
			desc := data["description"].(string)
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

			open, _ := os.Open("docs/swagger.json")
			defer open.Close()

			bys, _ := io.ReadAll(open)
			hm := make(map[string]interface{})
			json.Unmarshal(bys, &hm)

			apis := hm["paths"].(map[string]interface{})
			data := apis[reqURI].(map[string]interface{})[strings.ToLower(reqMethod)].(map[string]interface{})
			desc := data["description"].(string)

			exLog := entity.TExceptionLog{
				OptUri:        reqURI,
				OptMethod:     optFunc,
				RequestMethod: reqMethod,
				RequestParam:  string(reqData),
				OptDesc:       desc,
				ExceptionInfo: zlog.ErrorInfo,
				IpAddress:     ip,
				IpSource:      ipSource,
			}
			go repository.SaveExLog(exLog)
		}
	}
}
