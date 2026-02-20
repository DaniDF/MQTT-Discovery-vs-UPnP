package upnp

import (
	"crypto/rand"
	"fmt"
	"strings"
)

func FindHeader(header string, headerName string) (result string, flagFind bool) {
	for _, h := range strings.Split(header, "\n") {
		if strings.Contains(h, headerName) {
			h, _ = strings.CutPrefix(h, headerName+":")
			flagFind = true
			result = strings.Trim(h, " \t\r\n")
		}
	}

	return result, flagFind
}

func GenerateRandomUUID() (string, error) {
	buffer := make([]byte, 16) // See 1.1.4 128 bits -> 16 bytes
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("uuid:%x-%x-%x-%x-%x", buffer[:4], buffer[4:6], buffer[6:8], buffer[8:10], buffer[10:16]), nil
}
