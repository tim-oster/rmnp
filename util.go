// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net"
	"sync/atomic"
	"time"
)

func checkError(msg string, err error) {
	if err != nil {
		fmt.Println(msg, "=>", err)
		panic(err)
	}
}

func antiPanic(callback func()) {
	if err := recover(); err != nil {
		atomic.AddUint64(&StatGoRoutinePanics, 1)
		fmt.Println("panic:", err)

		if callback != nil {
			go func() {
				defer antiPanic(nil)
				callback()
			}()
		}
	}
}

func addrHash(addr *net.UDPAddr) uint32 {
	port := cnvUint32(uint32(addr.Port))
	return crc32.ChecksumIEEE(append(addr.IP, port...))
}

func cnvUint32(i uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, i)
	return b
}

func currentTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func greaterThanSequence(s1, s2 sequenceNumber) bool {
	return (s1 > s2 && s1-s2 <= 32768) || (s1 < s2 && s2-s1 > 32768)
}

func greaterThanOrder(s1, s2 orderNumber) bool {
	return (s1 > s2 && s1-s2 <= 127) || (s1 < s2 && s2-s1 > 127)
}

func differenceSequence(s1, s2 sequenceNumber) sequenceNumber {
	if s1 >= s2 {
		if s1-s2 <= 32768 {
			return s1 - s2
		}
		return (65535 - s1) + s2
	}
	return differenceSequence(s2, s1)
}

func min(x, y int64) int64 {
	if x < y {
		return x
	}

	return y
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}

	return y
}
