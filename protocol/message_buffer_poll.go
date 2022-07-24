package protocol

import "github.com/derekAHua/irpc/util"

var bufferPool = util.NewLimitedPool(512, 4096)

// PutData puts the byte slice into pool.
func PutData(data *[]byte) {
	bufferPool.Put(data)
}
