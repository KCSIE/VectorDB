package db

import (
	"fmt"
	"vectordb/model"
)

func QueryGetDBInfo() (model.ResDBInfo, error) {
	info, err := db.GetDBInfo()
	if err != nil {
		return model.ResDBInfo{}, err
	}

	return info, nil
}

func QueryCreateCollection(col *model.ReqCreateCollection) error {
	if _, ok := db.collections[col.Name]; ok {
		return fmt.Errorf("collection '%s' already exists", col.Name)
	}

	if col.Distance != "dot" && col.Distance != "cosine" && col.Distance != "euclidean" {
		return fmt.Errorf("invalid distance metric")
	}

	cfg := &model.CfgCollection{
		Dimension:   col.Dimension,
		IndexType:   col.IndexType,
		IndexParams: col.IndexParams,
		Distance:    col.Distance,
		Mapping:     col.Mapping,
	}

	if err := db.CreateCollection(col.Name, cfg); err != nil {
		return err
	}

	return nil
}

func QueryDeleteCollection(colname string) error {
	if _, ok := db.collections[colname]; !ok {
		return fmt.Errorf("collection '%s' not found", colname)
	}

	if err := db.DeleteCollection(colname); err != nil {
		return err
	}

	return nil
}

func QueryGetCollectionInfo(colname string) (model.ResCollectionInfo, error) {
	if _, ok := db.collections[colname]; !ok {
		return model.ResCollectionInfo{}, fmt.Errorf("collection '%s' not found", colname)
	}

	info, err := db.GetCollectionInfo(colname)
	if err != nil {
		return model.ResCollectionInfo{}, err
	}

	return info, nil
}

func QueryInsertObject(colname string, obj *model.ReqInsertObject) (string, error) {
	col, err := getCollection(colname)
	if err != nil {
		return "", err
	}
	if err := col.validateObjectMeta([]model.ReqInsertObject{*obj}); err != nil {
		return "", err
	}

	id, err := col.InsertObject(obj)
	if err != nil {
		return "", err
	}

	return id, nil
}

func QueryInsertObjects(colname string, objs *model.ReqInsertObjects) ([]string, error) {
	col, err := getCollection(colname)
	if err != nil {
		return nil, err
	}
	if err := col.validateObjectMeta(objs.Objects); err != nil {
		return nil, err
	}

	ids, err := col.InsertObjects(objs.Objects)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func QueryDeleteObject(colname string, objid string) error {
	col, err := getCollection(colname)
	if err != nil {
		return err
	}

	if err := col.DeleteObject(objid); err != nil {
		return err
	}

	return nil
}

func QueryUpdateObject(colname string, obj *model.ReqUpdateObject) error {
	col, err := getCollection(colname)
	if err != nil {
		return err
	}

	updateObj := model.ReqInsertObject{
		Metadata: obj.Metadata,
		Vector:   obj.Vector,
	}
	if err := col.validateObjectMeta([]model.ReqInsertObject{updateObj}); err != nil {
		return err
	}

	if err := col.UpdateObject(obj); err != nil {
		return err
	}

	return nil
}

func QueryGetObjects(colname string, offset int, limit int) ([]model.ResObjectInfo, error) {
	col, err := getCollection(colname)
	if err != nil {
		return nil, err
	}

	objs, err := col.GetObjects(offset, limit)
	if err != nil {
		return nil, err
	}

	return objs, nil
}

func QueryGetObjectInfo(colname string, objid string) (model.ResObjectInfo, error) {
	col, err := getCollection(colname)
	if err != nil {
		return model.ResObjectInfo{}, err
	}

	info, err := col.GetObjectInfo(objid)
	if err != nil {
		return model.ResObjectInfo{}, err
	}

	return info, nil
}

func QuerySearchObject(colname string, obj *model.ReqSearchObject) ([]model.ResSearchObject, error) {
	col, err := getCollection(colname)
	if err != nil {
		return nil, err
	}

	results, err := col.Search(obj.Vector, obj.TopK, obj.XParams)
	if err != nil {
		return nil, err
	}

	return results, nil
}
