package state

import "sync"

var UserStates = struct {
	sync.RWMutex
	M map[int64]string
}{M: make(map[int64]string)}
