package dto

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status string                 `json:"status" example:"ok"`
	Checks map[string]interface{} `json:"checks,omitempty"`
}

// SuccessResponse 通用成功响应
type SuccessResponse struct {
	Status  string      `json:"status" example:"ok"`
	Message string      `json:"message,omitempty" example:"操作成功"`
	Data    interface{} `json:"data,omitempty"`
}
