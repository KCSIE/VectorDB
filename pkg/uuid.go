package pkg

import "github.com/gofrs/uuid"

func NewUUID() (string, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.FromString(s)
}
