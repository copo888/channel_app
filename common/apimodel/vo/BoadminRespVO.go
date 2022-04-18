package vo

type BoadminRespVO struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Trace   string      `json:"trace"`
}