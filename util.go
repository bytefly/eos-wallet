package main

import (
	"strings"
)

func LeftShift(str string, size int) string {
	if str == "" || size <= 0 {
		return str
	}

	index := strings.IndexByte(str, '.')
	//for float type
	if index >= 0 {
		//drop dot(.)
		raw := []byte(str[:index])
		raw = append(raw, str[index+1:]...)
		if index > size { //move dot
			return str[:index-size] + "." + string(raw[index-size:])
		} else { //pad with 0.0s in prefix
			bytes := []byte("0.")
			for i := 0; i < size-index; i++ {
				bytes = append(bytes, '0')
			}
			bytes = append(bytes, raw...)
			return string(bytes)
		}
	}

	//for int
	if len(str) <= size {
		bytes := []byte("0.")
		for i := 0; i < size-len(str); i++ {
			bytes = append(bytes, '0')
		}
		bytes = append(bytes, []byte(str)...)
		return string(bytes)
	}

	return str[:len(str)-size] + "." + str[len(str)-size:]
}

func RightShift(str string, size int) string {
	if str == "" || size <= 0 {
		return str
	}

	index := strings.IndexByte(str, '.')
	//for int
	if index == -1 {
		bytes := []byte(str)
		for i := 0; i < size; i++ {
			bytes = append(bytes, '0')
		}
		return string(bytes)
	}

	//drop dot(.)
	bytes := []byte(str[:index])
	bytes = append(bytes, str[index+1:]...)
	if index+size >= len(str)-1 {
		for i := 0; i < index+size-len(str)+1; i++ {
			bytes = append(bytes, '0')
		}
	} else {
		bytes = append(bytes[:index+size], append([]byte("."), bytes[index+size:]...)...)
	}

	//trim all 0s in the head
	stop := -1
	for i := 0; i < len(bytes); i++ {
		if bytes[i] != '0' {
			stop = i
			break
		}
	}
	if stop >= 0 {
		if bytes[stop] == '.' {
			stop -= 1
		}
		bytes = bytes[stop:]
	}

	//trim all 0s in the tail
	stop = -1
	dot := strings.IndexByte(string(bytes), '.')
	if dot >= 0 {
		for i := len(bytes) - 1; i >= dot; i-- {
			if bytes[i] != '0' {
				stop = i
				break
			}
		}
		if stop >= 0 {
			if bytes[stop] == '.' {
				stop -= 1
			}
			bytes = bytes[0 : stop+1]
		}
	}
	return string(bytes)
}
