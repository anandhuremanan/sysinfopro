package main

import (
	"context"
	"sync"
)

// cancelSlot holds the one running instance of a long job (a disk scan, a
// search, a clean-up). Starting a new run cancels the previous one, and Cancel
// stops whatever is running. The zero value is ready to use.
type cancelSlot struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	run    uint64 // which run currently owns the slot
}

// start cancels any previous run and returns the new run's context; call done
// when the run finishes
func (s *cancelSlot) start(parent context.Context) (ctx context.Context, done func()) {
	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.run++
	mine := s.run
	s.cancel = cancel
	s.mu.Unlock()
	return ctx, func() {
		cancel()
		s.mu.Lock()
		if s.run == mine {
			s.cancel = nil
		}
		s.mu.Unlock()
	}
}

// stop cancels the running job, if any
func (s *cancelSlot) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}
