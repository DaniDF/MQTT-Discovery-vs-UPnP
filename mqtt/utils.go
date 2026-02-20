package mqtt

import (
	"crypto/rand"
	"fmt"
)

func GenerateID() (string, error) {
	buffer := make([]byte, 8)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x-%x-%x-%x", buffer[:2], buffer[2:4], buffer[4:6], buffer[6:8]), nil
}
