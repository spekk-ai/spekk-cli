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

// Dispatch queues msg and returns the worker the caller must start, which is
// only ever a worker this call claimed. accepted is false when the pool is at
// its cap.
//
// A message for a session that is already draining joins that worker's queue
// and starts nothing. Starting a second runner over one worker was enough to
// wedge the whole client: both runners released the same slot, the second
// release blocked on a channel already at capacity while holding the mutex,
// and every later dispatch then blocked on that mutex forever. The connection
// stayed up and the heartbeats kept flowing, so the sandbox looked healthy
// while accepting no work at all.
func (p *WorkerPool) Dispatch(msg Message) (start *Worker, accepted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Existing session - enqueue on its worker, which is still draining.
	if w, ok := p.sessions[msg.AgentSessionID]; ok {
		w.msgs <- msg
		return nil, true
	}

	// New session - try to claim a worker slot
	select {
	case <-p.work:
		// Got a slot
	default:
		return nil, false // at the cap
	}

	w := &Worker{
		agentSessionID: msg.AgentSessionID,
		msgs:           make(chan Message, 10),
	}
	p.sessions[msg.AgentSessionID] = w
	w.msgs <- msg
	return w, true
}

// finish releases the worker's slot when its queue is empty, and reports
// whether the worker is done.
//
// The emptiness test and the release share the mutex Dispatch holds, so a
// message queued at the last moment is never stranded by a runner that has
// already decided to stop. A runner that checked its queue outside the lock
// could return while Dispatch was enqueueing, and that message would then
// wait on a worker nobody was draining.
func (p *WorkerPool) finish(w *Worker) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(w.msgs) > 0 {
		return false
	}
	p.releaseLocked(w)
	return true
}

// Release returns a worker's slot. It covers the path where a runner leaves
// without finishing, a panic included.
func (p *WorkerPool) Release(w *Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.releaseLocked(w)
}

// releaseLocked returns the slot exactly once per claim. It releases only
// while this worker is the registered one, so a second call is a no-op
// rather than an extra token — an over-filled channel blocks the next
// release under the mutex and wedges the client.
func (p *WorkerPool) releaseLocked(w *Worker) {
	if cur, ok := p.sessions[w.agentSessionID]; !ok || cur != w {
		return
	}
	delete(p.sessions, w.agentSessionID)
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
