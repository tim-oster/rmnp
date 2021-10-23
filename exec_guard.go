package rmnp

import "sync"

type execGuard struct {
	mutex      sync.Mutex
	executions map[uint64]bool
}

func newExecGuard() *execGuard {
	guard := new(execGuard)
	guard.executions = make(map[uint64]bool)
	return guard
}

func (g *execGuard) tryExecute(id uint64) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if _, f := g.executions[id]; !f {
		g.executions[id] = true
		return true
	}

	return false
}

func (g *execGuard) finish(id uint64) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	delete(g.executions, id)
}
