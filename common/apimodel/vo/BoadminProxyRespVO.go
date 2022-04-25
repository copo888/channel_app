package vo

type BoadminProxyRespVO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
	traceId string `json:"traceId"`
}
