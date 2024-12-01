package model

type ReqCreateCollection struct {
	Name        string                 `json:"name" binding:"required"`
	Dimension   int                    `json:"dimension" binding:"required,gt=0"`
	IndexType   string                 `json:"index_type" binding:"required"`
	IndexParams map[string]interface{} `json:"index_params" binding:"required"`
	Distance    string                 `json:"dist_type" binding:"required"`
	Mapping     []string               `json:"mapping" binding:"required"`
}

type CfgCollection struct {
	Dimension   int                    `json:"dimension"`
	IndexType   string                 `json:"index_type"`
	IndexParams map[string]interface{} `json:"index_params"`
	Distance    string                 `json:"dist_type"`
	Mapping     []string               `json:"mapping"`
}

// todo: extra stats
type ResDBInfo struct {
	Collections     []string `json:"collections"`
	CollectionCount int      `json:"collection_count"`
}

// todo: extra stats
type ResCollectionInfo struct {
	Name        string                 `json:"name"`
	Dimension   int                    `json:"dimension"`
	IndexType   string                 `json:"index_type"`
	IndexParams map[string]interface{} `json:"index_params"`
	Distance    string                 `json:"dist_type"`
	Mapping     []string               `json:"mapping"`
	ObjectCount int                    `json:"object_count"`
}
