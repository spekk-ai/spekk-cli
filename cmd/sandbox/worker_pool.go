package main

import "sync"

const numWorkers = 5

type WorkerPool struct {
	work     chan struct{}
	sessions map[string]*Worker
	mu       sync.Mutex
}

func NewWorkerPool(size int) *WorkerPool {
	p := &WorkerPool{
		work:     make(chan struct{}, size),
		sessions: make(map[string]*Worker),
	}
	for i := 0; i < size; i++ {
		p.work <- struct{}{}
	}
	return p
}

func (p *WorkerPool) Dispatch(msg Message) *Worker {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Existing session - enqueue on its worker
	if w, ok := p.sessions[msg.AgentSessionID]; ok {
		w.msgs <- msg
		return w
	}

	// New session - try to claim a worker slot
	select {
	case <-p.work:
		// Got a slot
	default:
		return nil // at the cap
	}

	w := &Worker{
		agentSessionID: msg.AgentSessionID,
		msgs:           make(chan Message, 10),
	}
	p.sessions[msg.AgentSessionID] = w
	w.msgs <- msg
	return w
}

func (p *WorkerPool) Release(agentSessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, agentSessionID)
	p.work <- struct{}{}
}

func (p *WorkerPool) Cancel(agentSessionID string) {
	p.mu.Lock()
	w, ok := p.sessions[agentSessionID]
	p.mu.Unlock()
	if ok {
		w.Cancel()
	}
}
