// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

type ConnectionCallback func(*Connection)
type ConnectionCallbacks []ConnectionCallback

func invokeConnectionCallbacks(callbacks ConnectionCallbacks, connection *Connection) {
	for _, conn := range callbacks {
		conn(connection)
	}
}

func AddConnectionCallback(callbacks *ConnectionCallbacks, callback ConnectionCallback) {
	*callbacks = append(*callbacks, callback)
}
