// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"encoding/binary"
	"time"
	"fmt"
)

func checkError(msg string, err error) {
	if err != nil {
		fmt.Println(msg, "=>", err)
		panic(err)
	}
}

func cnvUint32(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

func currentTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func greaterThan(s1, s2 sequenceNumber) bool {
	return (s1 > s2 && s1 - s2 <= 32768) || (s1 < s2 && s2 - s1 > 32768)
}

func difference(s1, s2 sequenceNumber) sequenceNumber {
	if s1 >= s2 {
		if s1-s2 <= 32768 {
			return s1 - s2
		} else {
			return (65535 - s1) + s2
		}
	} else {
		return difference(s2, s1)
	}
}
