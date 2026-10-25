package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultMultipartMemory = 32 << 20

// QueryAllParams
//
//	@param ctx
//	@return map
func QueryAllParams(ctx *gin.Context) map[string]string {

	if ctx == nil {
		return map[string]string{}
	}

	query := ctx.Request.URL.Query()
	retData := make(map[string]string, len(query))
	for k, v := range query {
		retData[k] = v[0]
	}
	return retData
}

// AllGetParams ...
//
//	@param ctx
//	@return map
func AllGetParams(ctx *gin.Context) map[string]string {

	if ctx == nil {
		return map[string]string{}
	}

	querys := ctx.Request.URL.Query()
	retData := make(map[string]string, len(querys))
	for k, v := range querys {
		retData[k] = strings.Join(v, ",")
	}
	return retData
}

// AllPostParams  ...
//
//	@param ctx
//	@return map
func AllPostParams(ctx *gin.Context) map[string]interface{} {

	if ctx == nil {
		return map[string]interface{}{}
	}

	retData := map[string]interface{}{}
	contentType := ctx.ContentType()
	switch contentType {
	case gin.MIMEPOSTForm:
		ctx.Request.ParseForm()
		for k, v := range ctx.Request.Form {
			retData[k] = strings.Join(v, ",")
		}
	case gin.MIMEMultipartPOSTForm:
		if err := ctx.Request.ParseMultipartForm(defaultMultipartMemory); err == nil {
			for k, v := range ctx.Request.MultipartForm.Value {
				retData[k] = strings.Join(v, ",")
			}
			for k, v := range ctx.Request.MultipartForm.File {
				retData[k] = v
			}
		}
	default:
		if raw, err := ctx.GetRawData(); err == nil {
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(raw)) // 再写入body
			jsonDecoder := json.NewDecoder(bytes.NewReader(raw))
			jsonDecoder.UseNumber()
			if err := jsonDecoder.Decode(&retData); err != nil {
				retData["raw_data"] = raw
			}
		}
	}

	return retData
}

// AllHeader
//
//	@param ctx
//	@return map
func AllHeader(ctx *gin.Context) map[string]string {
	if ctx == nil {
		return map[string]string{}
	}

	retData := make(map[string]string)
	for k, v := range ctx.Request.Header {
		retData[k] = strings.Join(v, ",")
	}
	return retData
}

// Query
//
//	@param ctx
//	@param arge
//	@return string
func Query(ctx *gin.Context, arge ...string) string {
	for _, k := range arge {
		if val, e := ctx.GetQuery(k); e {
			return val
		}
	}
	return ""
}

// AllParams
//
//	@param ctx
//	@return map
func AllParams(ctx *gin.Context) map[string]interface{} {

	retData := map[string]interface{}{}
	for k, v := range AllGetParams(ctx) {
		retData[k] = v
	}

	if http.MethodPost == ctx.Request.Method {
		postData := AllPostParams(ctx)
		for k, v := range postData {
			retData[k] = v
		}
	}
	return retData
}

// Success  ...
//
//	@param ctx
//	@param data
func Success(ctx *gin.Context, data interface{}) {
	if ctx.IsAborted() {
		return
	}
	ctx.PureJSON(http.StatusOK, &JSONResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Failure ...
//
//	@param ctx
//	@param code
//	@param message
func Failure(ctx *gin.Context, code int64, message string) {
	if ctx.IsAborted() {
		return
	}
	ctx.PureJSON(http.StatusOK, &JSONResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

type JSONResponse struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
