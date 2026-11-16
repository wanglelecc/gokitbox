package socketserver

import (
	"sync/atomic"
	"time"
)

// Stats contains process-local gauges and cumulative counters. Snapshot fields
// are individually concurrency-safe, not one transaction across all resources.
type Stats struct {
	Connections         int
	PendingConnections  int
	ReadBytes           int64
	WriteBytes          int64
	MaintenancePending  int
	MaintenanceCalls    uint64
	MaintenanceErrors   uint64
	MaintenanceMaxDelay time.Duration
}

func (s *Server) Stats() Stats {
	s.mu.Lock()
	pending := s.admitting
	s.mu.Unlock()
	s.maintenanceMu.Lock()
	waiting := len(s.maintenanceHeap)
	s.maintenanceMu.Unlock()
	return Stats{Connections: s.hub.Count(), PendingConnections: pending, ReadBytes: s.readBytes.Load(), WriteBytes: s.writeBytes.Load(), MaintenancePending: waiting, MaintenanceCalls: s.maintenanceCalls.Load(), MaintenanceErrors: s.maintenanceErrors.Load(), MaintenanceMaxDelay: time.Duration(s.maintenanceMaxDelay.Load())}
}
func reserveBytes(counter *atomic.Int64, limit, size int64) bool {
	for {
		current := counter.Load()
		if limit > 0 && size > limit-current {
			return false
		}
		if counter.CompareAndSwap(current, current+size) {
			return true
		}
	}
}
