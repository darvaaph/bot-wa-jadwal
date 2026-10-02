package maintenance

import (
	"sync"
	"sync/atomic"
)

var restoreLock sync.Mutex
var mutationLock sync.RWMutex
var restoring atomic.Bool

// EnterMutation keeps a request from overlapping a restore. A false result
// means the caller should return a retryable maintenance response.
func EnterMutation() (func(), bool) {
	if restoring.Load() {
		return nil, false
	}
	mutationLock.RLock()
	if restoring.Load() {
		mutationLock.RUnlock()
		return nil, false
	}
	return mutationLock.RUnlock, true
}

func BeginRestore() func() {
	restoreLock.Lock()
	restoring.Store(true)
	mutationLock.Lock()
	return func() {
		mutationLock.Unlock()
		restoring.Store(false)
		restoreLock.Unlock()
	}
}

func Restoring() bool { return restoring.Load() }
