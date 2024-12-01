package model

type ReqInsertObject struct {
	Metadata map[string]interface{} `json:"metadata" binding:"required"`
	Vector   []float32              `json:"vector" binding:"required"`
}

type ReqInsertObjects struct {
	Objects []ReqInsertObject `json:"objects" binding:"required"`
}

type ReqUpdateObject struct {
	ID       string                 `json:"id" binding:"required"`
	Metadata map[string]interface{} `json:"metadata" binding:"required"`
	Vector   []float32              `json:"vector" binding:"required"`
}

type ReqGetObjects struct {
	Offset int `json:"offset" binding:"gte=0"`
	Limit  int `json:"limit" binding:"gte=1"`
}

type ReqSearchObject struct {
	Vector  []float32              `json:"vector" binding:"required"`
	TopK    int                    `json:"topk" binding:"required"`
	XParams map[string]interface{} `json:"x_params" binding:"omitempty"`
}

type ResObjectInfo struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
	Vector   []float32              `json:"vector"`
}

type ResSearchObject struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
	Vector   []float32              `json:"vector"`
	Score    float32                `json:"score"`
}
