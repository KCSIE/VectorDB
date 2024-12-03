package index

import (
	"fmt"
	"vectordb/db/index/flat"
	"vectordb/db/index/hnsw"
	"vectordb/model"
)

// todo: snapshot to load and save index?
type Indexer interface {
	Insert(id string, vector []float32) error
	Delete(id string) error
	Update(id string, vector []float32) error
	Search(vector []float32, topk int, xparams map[string]interface{}) ([]model.SearchResult, error)
}

func NewIndexer(cfg *model.CfgCollection) (Indexer, error) {
	switch cfg.IndexType {
	case "flat":
		params, err := model.ValidateAndConvert(cfg.IndexType, cfg.IndexParams)
		if err != nil {
			return nil, err
		}
		idx, err := flat.NewFlat(params.(*model.FlatParams), cfg.Distance)
		if err != nil {
			return nil, err
		}
		return idx, nil
	case "hnsw":
		params, err := model.ValidateAndConvert(cfg.IndexType, cfg.IndexParams)
		if err != nil {
			return nil, err
		}
		idx, err := hnsw.NewHNSW(params.(*model.HNSWParams), cfg.Distance)
		if err != nil {
			return nil, err
		}
		return idx, nil
	default:
		return nil, fmt.Errorf("unsupported index type: '%s'", cfg.IndexType)
	}
}
