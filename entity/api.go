package entity

type HttpResponseObject struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	ErrCode int         `json:"err_code"`
	Data    interface{} `json:"data"`
}
