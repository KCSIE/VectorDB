package pkg

import (
	"bytes"
	"encoding/gob"
)

func Serialize(s interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Deserialize(data []byte, s interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(s); err != nil {
		return err
	}
	return nil
}
