package main

import (
	"strconv"
)

func IntToHex(num int64) []byte {
	return []byte(strconv.FormatInt(num, 16))
}

func ReverseBytes(data []byte) {
	for i, j := 0, len(data) -1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
