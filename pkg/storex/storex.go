package storex

import (
	"sync"
)

var (
	reqData sync.Map
)

// 获取当前 Goroutine ID
func goid() uint64 {
	var buf [64]byte
	n := 0
	for _, c := range buf {
		if c == ' ' {
			n++
			continue
		}
		if c == '\n' {
			break
		}
		if n >= 1 {
			break
		}
	}
	var gid uint64
	for _, c := range buf[n:] {
		if c < '0' || c > '9' {
			break
		}
		gid = gid*10 + uint64(c-'0')
	}
	return gid
}

// Set 存储数据（使用空结构体做唯一 Key）
func Set(key any, val any) {
	gid := goid()
	if m, ok := reqData.Load(gid); ok {
		m.(*sync.Map).Store(key, val)
	} else {
		var m sync.Map
		m.Store(key, val)
		reqData.Store(gid, &m)
	}
}

// Get 获取数据
func Get(key any) (any, bool) {
	gid := goid()
	if m, ok := reqData.Load(gid); ok {
		return m.(*sync.Map).Load(key)
	}
	return nil, false
}

// Clean 清空当前请求数据
func Clean() {
	gid := goid()
	reqData.Delete(gid)
}
