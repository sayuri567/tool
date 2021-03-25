package api

const (
	SUCCESS      = "success"
	UNAUTHORIZED = "unauthorized"
	NOT_FOUND    = "not found"
)

// Output http请求response
type Output struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
