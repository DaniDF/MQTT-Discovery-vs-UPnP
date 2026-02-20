package utils

import (
	"net"
	"strings"
	"time"
)

func GetLocalIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func AlertAfter(timeout time.Duration, channel chan bool) {
	go func() {
		time.Sleep(timeout)
		channel <- true
	}()
}

func Find[T any](elements []T, filter func(T) bool) []T {
	var result []T

	for _, element := range elements {
		if filter(element) {
			result = append(result, element)
		}
	}

	return result
}

func FindFirst[T any](elements []T, filter func(T) bool) (T, bool) {
	var result T

	findResult := Find(elements, filter)
	if len(findResult) == 0 {
		return result, false
	}

	return findResult[0], true
}

func StringToCSV(slice []string) string {
	var result strings.Builder

	for i, element := range slice {
		if i != 0 {
			result.WriteString(",")
		}
		result.WriteString(element)
	}

	return result.String()
}

type EqualComparable[T any] interface {
	Equal(T) bool
}

// Deletes all the oppurence of element in slice
func DeleteElement[T EqualComparable[T]](slice []T, element T) []T {
	var result []T
	for _, el := range slice {
		if el.Equal(element) {
			result = append(result, el)
		}
	}

	return result
}
