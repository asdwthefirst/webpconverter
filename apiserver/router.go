package apiserver

import (
	"github.com/gin-gonic/gin"
	"webpconverter/entity"
	fc "webpconverter/feedconst"
)

type ApiController struct {
	*gin.Engine
}

func NewApiController(r *gin.Engine) *ApiController {
	return &ApiController{r}
}

func (router *ApiController) Router() {

	//做sticker图片转化
	router.Handle("POST", "/sticker/transform", Transform)

}

func HttpSuccessResp(ctx *gin.Context, httpCode int, msg string, data interface{}) {
	resp := entity.HttpResponseObject{
		Status:  fc.HttpSuccessful,
		Message: msg,
		ErrCode: fc.DefaultErrCode,
		Data:    data,
	}

	ctx.JSON(httpCode, resp)
}

func HttpFailResp(ctx *gin.Context, httpCode int, errCode int, data interface{}) {
	resp := entity.HttpResponseObject{
		Status:  fc.HttpFailure,
		Message: fc.ErrorHandler[errCode],
		ErrCode: errCode,
		Data:    data,
	}
	ctx.JSON(httpCode, resp)
}
