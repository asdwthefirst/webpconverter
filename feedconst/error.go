package feedconst

//http status字段
const (
	HttpSuccessful = 1
	HttpFailure    = 0
)

//http接口err_code和message提示信息
const (
	DefaultErrCode = 0

	//1000-param err

	RequestParamErrCode       = 1000
	RequestParamErrMsg        = "params err"
	RequestParamStructErrCode = 1001
	RequestParamStructErrMsg  = "params err:invalid data"
	RequestFormDataErrCode    = 1002
	RequestFormDataErrMsg     = "get form data err"

	TransformFailCode = 6000
	TransformFailMsg  = "fail to transform"
)

var ErrorHandler = map[int]string{
	RequestParamErrCode:       RequestParamErrMsg,
	RequestParamStructErrCode: RequestParamStructErrMsg,
	RequestFormDataErrCode:    RequestFormDataErrMsg,
	TransformFailCode:         TransformFailMsg,
}
