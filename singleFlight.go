package cache

import "sync"

type Call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type RequestGroup struct {
	mu    sync.Mutex // protects calls
	calls map[string]*Call
}

// CallOnce Make sure that fn will only be executed once at a time
func (g *RequestGroup) CallOnce(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[string]*Call)
	}
	if call, ok := g.calls[key]; ok {
		g.mu.Unlock()
		call.wg.Wait()
		return call.val, call.err
	}
	c := new(Call)
	c.wg.Add(1)
	g.calls[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()

	return c.val, c.err
}
