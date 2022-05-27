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

	//2000-permission err

	ForbidGetS3InfoErrCode  = 2000
	ForbidGetS3InfoErrMsg   = "forbid get s3 info"
	GetYtmp3RankDataErrCode = 2001
	GetYtmp3RankDataErrMsg  = "get ytmp3 data err"
	RequestYtmp3HttpErrCode = 2002
	RequestYtmp3HttpErrMsg  = "request ytmp3 http err"
	NewYtmp3HttpReqErrCode  = 2003
	NewYtmp3HttpReqErrMsg   = "ytmp3 NewRequest err"

	//3000-db mysql error

	RedisServerErrorCode      = 3000
	RedisServerErrorMsg       = "redis server abnormal"
	RedisServerExecErrorCode  = 3001
	RedisServerExecErrorMsg   = "redis exec fail"
	RedisServerExecScriptCode = 3002
	RedisServerExecScriptMSg  = "redis exec fail"

	MysqlExecErrorCode = 4000
	MysqlExecErrorMsg  = "mysql exec fail"

	ResourceNotExistCode      = 5000
	ResourceNotExistMsg       = "resource not exist"
	ActivityAlreadyEndingCode = 5001
	ActivityAlreadyEnding     = "activity already end"
	ActivityNotStartCode      = 5002
	ActivityNotStartMsg       = "activity not start"

	TransformFailCode = 6000
	TransformFailMsg  = "fail to transform"
)

var ErrorHandler = map[int]string{
	ForbidGetS3InfoErrCode:    ForbidGetS3InfoErrMsg,
	RequestParamErrCode:       RequestParamErrMsg,
	RedisServerErrorCode:      RedisServerErrorMsg,
	RedisServerExecErrorCode:  RedisServerExecErrorMsg,
	RequestParamStructErrCode: RequestParamStructErrMsg,
	GetYtmp3RankDataErrCode:   GetYtmp3RankDataErrMsg,
	RequestYtmp3HttpErrCode:   RequestYtmp3HttpErrMsg,
	NewYtmp3HttpReqErrCode:    NewYtmp3HttpReqErrMsg,
	MysqlExecErrorCode:        MysqlExecErrorMsg,
	ResourceNotExistCode:      ResourceNotExistMsg,
	RedisServerExecScriptCode: RedisServerExecScriptMSg,
	ActivityAlreadyEndingCode: ActivityAlreadyEnding,
	ActivityNotStartCode:      ActivityNotStartMsg,
	RequestFormDataErrCode:    RequestFormDataErrMsg,
	TransformFailCode:         TransformFailMsg,
}
