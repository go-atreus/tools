package apiresp

type ApiResponse struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
	ErrDlt  string `json:"errDlt"`
	Data    any    `json:"data,omitempty"`
}

func ApiSuccess(data any) *ApiResponse {
	if format, ok := data.(ApiFormat); ok {
		format.ApiFormat()
	}
	return &ApiResponse{
		Data: data,
	}
}

func ParseError(err error) *ApiResponse {
	if err == nil {
		return ApiSuccess(nil)
	}
	//unwrap := errs.Unwrap(err)
	//if codeErr, ok := unwrap.(errs.CodeError); ok {
	//	resp := ApiResponse{ErrCode: codeErr.Code(), ErrMsg: codeErr.Msg(), ErrDlt: codeErr.Detail()}
	//	if resp.ErrDlt == "" {
	//		resp.ErrDlt = err.Error()
	//	}
	//	return &resp
	//}
	return &ApiResponse{ErrCode: 1000, ErrMsg: err.Error()}
}
