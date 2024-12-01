package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"vectordb/model"
	"vectordb/pkg"

	"go.etcd.io/bbolt"
)

var db *DB

const (
	bucketCollectionsMetadata = "collections_metadata"
	bucketCollectionObjects   = "collection_objects"
)

type DB struct {
	collections map[string]*Collection
	kv          *bbolt.DB
	mu          sync.RWMutex
	path        string
}

func Init(path string) (err error) {
	// set default path
	if path == "" {
		path = "./vectordb_data"
	}

	// if the path doesn't exist
	_, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(path, 0750)
			if err != nil {
				return fmt.Errorf("couldn't create store path '%s': %w", path, err)
			}
		} else {
			return fmt.Errorf("couldn't get info about store path '%s': %w", path, err)
		}
	}

	db = &DB{
		collections: make(map[string]*Collection),
		path:        path,
	}
	if err = db.NewDB(path); err != nil {
		return fmt.Errorf("failed to load db: %w", err)
	}
	return nil
}

func Close() {
	_ = db.Close()
}

func (db *DB) NewDB(path string) (err error) {
	kvpath := filepath.Join(path, "vectordb.db")
	db.kv, err = bbolt.Open(kvpath, 0600, nil)
	if err != nil {
		return fmt.Errorf("failed to open kv db: %w", err)
	}

	// load collections and metadata
	db.collections = map[string]*Collection{}

	if err := db.kv.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketCollectionsMetadata))
		if err != nil {
			return fmt.Errorf("failed to create or load bucket: %w", err)
		}
		b.ForEach(func(k, v []byte) error {
			colmeta := &model.CfgCollection{}
			if err := pkg.Deserialize(v, colmeta); err != nil {
				return fmt.Errorf("failed to deserialize collection: %w", err)
			}
			col, err := newCollection(string(k), colmeta)
			if err != nil {
				return fmt.Errorf("failed to new collection instance: %w", err)
			}
			db.collections[string(k)] = col
			return nil
		})
		return nil
	}); err != nil {
		return fmt.Errorf("failed to load collections: %w", err)
	}

	return nil
}

func (db *DB) Close() (err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, col := range db.collections {
		if err = col.Close(); err != nil {
			return fmt.Errorf("failed to close collection '%s': %w", col.name, err)
		}
	}

	if err = db.kv.Close(); err != nil {
		return fmt.Errorf("failed to close kv db: %w", err)
	}

	return nil
}

func (db *DB) CreateCollection(colname string, cfg *model.CfgCollection) (err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	col, err := newCollection(colname, cfg)
	if err != nil {
		return fmt.Errorf("failed to new collection instance: %w", err)
	}
	db.collections[colname] = col

	if err := db.kv.Update(func(tx *bbolt.Tx) error {
		colmeta, err := pkg.Serialize(col.config)
		if err != nil {
			return fmt.Errorf("failed to serialize collection: %w", err)
		}

		metaBucket := tx.Bucket([]byte(bucketCollectionsMetadata))
		if err := metaBucket.Put([]byte(colname), colmeta); err != nil {
			return fmt.Errorf("failed to put collection metadata: %w", err)
		}

		colBucket, err := tx.CreateBucket([]byte(colname))
		if err != nil {
			return fmt.Errorf("failed to create collection bucket: %w", err)
		}
		_, err = colBucket.CreateBucket([]byte(bucketCollectionObjects))
		if err != nil {
			return fmt.Errorf("failed to create object bucket under collection '%s': %w", colname, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create collection '%s': %w", colname, err)
	}

	return nil
}

func (db *DB) DeleteCollection(name string) (err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if col, ok := db.collections[name]; ok {
		if err := col.Close(); err != nil {
			return fmt.Errorf("failed to close collection WAL: %w", err)
		}
	}
	walpath := filepath.Join(db.path, name+".wal")
	if err := os.RemoveAll(walpath); err != nil {
		return fmt.Errorf("failed to delete WAL directory: %w", err)
	}

	if err := db.kv.Update(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket([]byte(bucketCollectionsMetadata))
		if err := metaBucket.Delete([]byte(name)); err != nil {
			return fmt.Errorf("failed to delete collection metadata: %w", err)
		}

		if err := tx.DeleteBucket([]byte(name)); err != nil {
			return fmt.Errorf("failed to delete collection bucket: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete collection '%s': %w", name, err)
	}

	delete(db.collections, name)

	return nil
}

func (db *DB) GetCollectionInfo(colname string) (model.ResCollectionInfo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	cnt := 0
	if err := db.kv.View(func(tx *bbolt.Tx) error {
		colBucket := tx.Bucket([]byte(colname))
		if colBucket == nil {
			return fmt.Errorf("bucket for collection '%s' not found", colname)
		}
		objectBucket := colBucket.Bucket([]byte(bucketCollectionObjects))
		if objectBucket == nil {
			return fmt.Errorf("object bucket for collection '%s' not found", colname)
		}
		cnt = objectBucket.Stats().KeyN
		return nil
	}); err != nil {
		return model.ResCollectionInfo{}, fmt.Errorf("failed to get collection '%s' info: %w", colname, err)
	}

	// todo: get extra stats

	info := model.ResCollectionInfo{
		Name:        colname,
		Dimension:   db.collections[colname].config.Dimension,
		IndexType:   db.collections[colname].config.IndexType,
		IndexParams: db.collections[colname].config.IndexParams,
		Distance:    db.collections[colname].config.Distance,
		Mapping:     db.collections[colname].config.Mapping,
		ObjectCount: cnt,
	}

	return info, nil
}

func (db *DB) GetDBInfo() (model.ResDBInfo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	cols := []string{}
	for colname := range db.collections {
		cols = append(cols, colname)
	}

	// todo: get extra stats

	info := model.ResDBInfo{
		Collections:     cols,
		CollectionCount: len(db.collections),
	}

	return info, nil
}
