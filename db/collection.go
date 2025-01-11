package db

import (
	"fmt"
	"path/filepath"
	"sync"
	"vectordb/db/index"
	"vectordb/model"
	"vectordb/pkg"

	"github.com/tidwall/wal"
	"go.etcd.io/bbolt"
)

type WALEntryType int

const (
	WALInsert WALEntryType = iota
	WALDelete
	WALUpdate
)

type WALEntry struct {
	Type   WALEntryType
	ID     string
	Vector []float32
}

type Collection struct {
	name   string
	config model.CfgCollection
	index  index.Indexer
	mu     sync.RWMutex
	wal    *wal.Log
	seq    uint64
}

func newCollection(colname string, cfg *model.CfgCollection) (*Collection, error) {
	col := Collection{
		name:   colname,
		config: *cfg,
	}

	walPath := filepath.Join(db.path, colname+".wal")
	log, err := wal.Open(walPath, &wal.Options{
		NoSync: false,
		NoCopy: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL: %w", err)
	}
	col.wal = log

	idx, err := index.NewIndexer(cfg)
	if err != nil {
		return nil, err
	}
	col.index = idx

	if err := col.replayWAL(); err != nil {
		return nil, fmt.Errorf("failed to replay WAL: %w", err)
	}

	return &col, nil
}

func (c *Collection) replayWAL() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	first, err := c.wal.FirstIndex()
	if err != nil {
		return err
	}
	last, err := c.wal.LastIndex()
	if err != nil {
		return err
	}

	c.seq = last

	for i := first; i <= last; i++ {
		data, err := c.wal.Read(i)
		if err != nil {
			continue
		}

		var entry WALEntry
		if err := pkg.Deserialize(data, &entry); err != nil {
			continue
		}

		switch entry.Type {
		case WALInsert:
			c.index.Insert(entry.ID, entry.Vector)
		case WALDelete:
			c.index.Delete(entry.ID)
		case WALUpdate:
			c.index.Update(entry.ID, entry.Vector)
		}
	}
	return nil
}

func getCollection(colname string) (*Collection, error) {
	col, ok := db.collections[colname]
	if !ok {
		return nil, fmt.Errorf("collection '%s' not found", colname)
	}

	return col, nil
}

func (c *Collection) validateObjectMeta(objs []model.ReqInsertObject) error {
	for i, obj := range objs {
		if len(obj.Metadata) != len(c.config.Mapping) {
			return fmt.Errorf("metadata length mismatch in object %d", i)
		}

		if len(obj.Vector) != c.config.Dimension {
			return fmt.Errorf("vector dimension mismatch in object %d", i)
		}

		for _, key := range c.config.Mapping {
			if _, ok := obj.Metadata[key]; !ok {
				return fmt.Errorf("metadata key '%s' not found in object %d", key, i)
			}
		}
	}
	return nil
}

func (c *Collection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.wal != nil {
		if err := c.wal.Close(); err != nil {
			return fmt.Errorf("failed to close WAL: %w", err)
		}
	}
	return nil
}

func (c *Collection) insertObject(tx *bbolt.Tx, obj *model.ReqInsertObject) (string, error) {
	id, err := pkg.NewUUID()
	if err != nil {
		return "", err
	}

	entry := WALEntry{
		Type:   WALInsert,
		ID:     id,
		Vector: obj.Vector,
	}
	walData, err := pkg.Serialize(entry)
	if err != nil {
		return "", fmt.Errorf("failed to serialize WAL entry: %w", err)
	}

	c.seq++
	if err := c.wal.Write(c.seq, walData); err != nil {
		return "", fmt.Errorf("failed to write to WAL: %w", err)
	}

	colBucket := tx.Bucket([]byte(c.name))
	objBucket := colBucket.Bucket([]byte(bucketCollectionObjects))

	objBytes, err := pkg.Serialize(obj)
	if err != nil {
		return "", fmt.Errorf("failed to serialize object: %w", err)
	}

	if err := objBucket.Put([]byte(id), objBytes); err != nil {
		return "", fmt.Errorf("failed to put object under collection '%s': %w", c.name, err)
	}

	if err := c.index.Insert(id, obj.Vector); err != nil {
		return "", err
	}

	return id, nil
}

func (c *Collection) InsertObject(obj *model.ReqInsertObject) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var id string
	if err := db.kv.Update(func(tx *bbolt.Tx) error {
		var err error
		id, err = c.insertObject(tx, obj)
		return err
	}); err != nil {
		return "", fmt.Errorf("failed to insert object into collection '%s': %w", c.name, err)
	}

	return id, nil
}

func (c *Collection) InsertObjects(objs []model.ReqInsertObject) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := len(objs)
	type result struct {
		pos int
		id  string
		err error
	}
	ch := make(chan result, n)
	var wg sync.WaitGroup
	ids := make([]string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var id string
			err := db.kv.Batch(func(tx *bbolt.Tx) error {
				var err error
				id, err = c.insertObject(tx, &objs[i])
				return err
			})
			ch <- result{pos: i, id: id, err: err}
		}(i)
	}

	wg.Wait()
	close(ch)

	for res := range ch {
		if res.err != nil {
			return nil, res.err
		}
		ids[res.pos] = res.id
	}

	return ids, nil
}

func (c *Collection) DeleteObject(objid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := WALEntry{
		Type: WALDelete,
		ID:   objid,
	}
	walData, err := pkg.Serialize(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize WAL entry: %w", err)
	}

	c.seq++
	if err := c.wal.Write(c.seq, walData); err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	if err := db.kv.Update(func(tx *bbolt.Tx) error {
		colBucket := tx.Bucket([]byte(c.name))
		objBucket := colBucket.Bucket([]byte(bucketCollectionObjects))

		if err := objBucket.Delete([]byte(objid)); err != nil {
			return fmt.Errorf("failed to delete object under collection '%s': %w", c.name, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete object from collection '%s': %w", c.name, err)
	}

	if err := c.index.Delete(objid); err != nil {
		return err
	}

	return nil
}

func (c *Collection) UpdateObject(obj *model.ReqUpdateObject) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := WALEntry{
		Type:   WALUpdate,
		ID:     obj.ID,
		Vector: obj.Vector,
	}
	walData, err := pkg.Serialize(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize WAL entry: %w", err)
	}

	c.seq++
	if err := c.wal.Write(c.seq, walData); err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	if err := db.kv.Update(func(tx *bbolt.Tx) error {
		colBucket := tx.Bucket([]byte(c.name))
		objBucket := colBucket.Bucket([]byte(bucketCollectionObjects))

		if exist := objBucket.Get([]byte(obj.ID)); exist == nil {
			return fmt.Errorf("object %s not found", obj.ID)
		}

		objBytes, err := pkg.Serialize(obj)
		if err != nil {
			return fmt.Errorf("failed to serialize object: %w", err)
		}

		if err := objBucket.Put([]byte(obj.ID), objBytes); err != nil {
			return fmt.Errorf("failed to update object under collection '%s': %w", c.name, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to update object in collection '%s': %w", c.name, err)
	}

	if err := c.index.Update(obj.ID, obj.Vector); err != nil {
		return err
	}

	return nil
}

func (c *Collection) GetObjects(offset int, limit int) ([]model.ResObjectInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	objs := []model.ResObjectInfo{}

	if err := db.kv.View(func(tx *bbolt.Tx) error {
		colBucket := tx.Bucket([]byte(c.name))
		objBucket := colBucket.Bucket([]byte(bucketCollectionObjects))

		cursor := objBucket.Cursor()
		skipped, fetched := 0, 0

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if skipped < offset {
				skipped++
				continue
			}

			if fetched < limit {
				obj := new(model.ReqInsertObject)
				if err := pkg.Deserialize(v, obj); err != nil {
					return fmt.Errorf("failed to deserialize object: %w", err)
				}

				objs = append(objs, model.ResObjectInfo{
					ID:       string(k),
					Metadata: obj.Metadata,
					Vector:   obj.Vector,
				})
				fetched++
			} else {
				break
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get objects from collection '%s': %w", c.name, err)
	}

	return objs, nil
}

func (c *Collection) GetObjectInfo(objid string) (model.ResObjectInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := model.ResObjectInfo{}

	if err := db.kv.View(func(tx *bbolt.Tx) error {
		colBucket := tx.Bucket([]byte(c.name))
		objBucket := colBucket.Bucket([]byte(bucketCollectionObjects))

		objBytes := objBucket.Get([]byte(objid))
		if objBytes == nil {
			return fmt.Errorf("object %s not found", objid)
		}

		obj := new(model.ReqInsertObject)
		if err := pkg.Deserialize(objBytes, obj); err != nil {
			return fmt.Errorf("failed to deserialize object: %w", err)
		}

		res = model.ResObjectInfo{
			ID:       string(objid),
			Metadata: obj.Metadata,
			Vector:   obj.Vector,
		}

		return nil
	}); err != nil {
		return model.ResObjectInfo{}, fmt.Errorf("failed to get object info from collection '%s': %w", c.name, err)
	}

	return res, nil
}

func (c *Collection) Search(vector []float32, topk int, xparams map[string]interface{}) ([]model.ResSearchObject, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results, err := c.index.Search(vector, topk, xparams)
	if err != nil {
		return nil, err
	}

	res := []model.ResSearchObject{}

	for _, result := range results {
		obj, err := c.GetObjectInfo(result.ID)
		if err != nil {
			return nil, err
		}

		res = append(res, model.ResSearchObject{
			ID:       obj.ID,
			Metadata: obj.Metadata,
			Vector:   obj.Vector,
			Score:    result.Score,
		})
	}

	return res, nil
}
