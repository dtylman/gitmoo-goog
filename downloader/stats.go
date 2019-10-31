package downloader

import (
	"sync"
)

// Stats TODO
type Stats struct {
	Total int
	Errors int
	TotalSize uint64
	Downloaded int

	mutex sync.Mutex
}

// UpdateStatsTotal TODO
func (s *Stats) UpdateStatsTotal(total int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Total += total
}

// UpdateStatsDownloaded TODO
func (s *Stats) UpdateStatsDownloaded(totalSize uint64, downloaded int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.TotalSize += totalSize
	s.Downloaded += downloaded
}

// UpdateStatsError TODO
func (s *Stats) UpdateStatsError(errors int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Errors += errors
}