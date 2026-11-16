package socketserver

import (
	"container/heap"
	"context"
	"math/rand/v2"
	"time"
)

// The indexed heap holds each waiting session once. Workers are bounded but
// pending work is never discarded when every worker is busy.
type maintenanceHeap []*Session

func (h maintenanceHeap) Len() int           { return len(h) }
func (h maintenanceHeap) Less(i, j int) bool { return h[i].maintenanceDue.Before(h[j].maintenanceDue) }
func (h maintenanceHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].maintenanceIndex = i
	h[j].maintenanceIndex = j
}
func (h *maintenanceHeap) Push(v any) {
	s := v.(*Session)
	s.maintenanceIndex = len(*h)
	*h = append(*h, s)
}
func (h *maintenanceHeap) Pop() any {
	old := *h
	n := len(old)
	s := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	s.maintenanceIndex = -1
	return s
}

func (s *Server) scheduleMaintenance(session *Session, first bool) {
	interval := s.options.MaintenanceInterval
	if interval <= 0 || s.settings.hooks.OnMaintain == nil {
		return
	}
	// First renewals spread across half an interval; subsequent renewals jitter
	// by up to 10 percent earlier, never later than the configured interval.
	span := interval / 10
	if first {
		span = interval / 2
	}
	delay := interval
	if span > 0 {
		delay -= time.Duration(rand.Int64N(int64(span)))
	}
	s.maintenanceMu.Lock()
	if !session.Closed() && !session.draining.Load() && session.maintenanceIndex < 0 {
		session.maintenanceDue = time.Now().Add(delay)
		heap.Push(&s.maintenanceHeap, session)
	}
	s.maintenanceMu.Unlock()
	s.wakeMaintenance()
}
func (s *Server) wakeMaintenance() {
	select {
	case s.maintenanceWake <- struct{}{}:
	default:
	}
}
func (s *Server) unscheduleMaintenance(session *Session) {
	s.maintenanceMu.Lock()
	if session.maintenanceIndex >= 0 {
		heap.Remove(&s.maintenanceHeap, session.maintenanceIndex)
	}
	s.maintenanceMu.Unlock()
	s.wakeMaintenance()
}
func (s *Server) dispatchMaintenance() {
	defer s.workersWG.Done()
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for {
		s.maintenanceMu.Lock()
		var session *Session
		delay := time.Hour
		if len(s.maintenanceHeap) > 0 {
			delay = time.Until(s.maintenanceHeap[0].maintenanceDue)
			if delay <= 0 {
				session = heap.Pop(&s.maintenanceHeap).(*Session)
			}
		}
		s.maintenanceMu.Unlock()
		if session != nil {
			select {
			case s.maintenance <- session:
			case <-s.runtimeCtx.Done():
				return
			}
			continue
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(delay)
		select {
		case <-s.runtimeCtx.Done():
			return
		case <-s.maintenanceWake:
		case <-timer.C:
		}
	}
}
func (s *Server) maintenanceWorker() {
	defer s.workersWG.Done()
	for {
		select {
		case <-s.runtimeCtx.Done():
			return
		case session := <-s.maintenance:
			if !session.Closed() && !session.draining.Load() {
				delay := time.Since(session.maintenanceDue)
				for old := s.maintenanceMaxDelay.Load(); int64(delay) > old; old = s.maintenanceMaxDelay.Load() {
					if s.maintenanceMaxDelay.CompareAndSwap(old, int64(delay)) {
						break
					}
				}
				err := session.hook(func(ctx context.Context) error { return s.settings.hooks.OnMaintain(ctx, session) })
				s.maintenanceCalls.Add(1)
				if err != nil {
					s.maintenanceErrors.Add(1)
					s.report(session, "maintain", err)
				}
				s.scheduleMaintenance(session, false)
			}
		}
	}
}
