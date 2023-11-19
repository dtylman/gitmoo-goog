package downloader

import (
	"sync"
)

// Stats TODO
type Stats struct {
	Total      int
	Errors     int
	TotalSize  uint64
	Downloaded int
	Skipped    int

	mutex sync.Mutex
}

// UpdateStatsTotal increment the total items
func (s *Stats) UpdateStatsTotal(total int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Total += total
}

// UpdateStatsDownloaded increment the downloaded items and size of items
func (s *Stats) UpdateStatsDownloaded(totalSize uint64, downloaded int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.TotalSize += totalSize
	s.Downloaded += downloaded
}

// UpdateStatsError increment the items that produced errors
func (s *Stats) UpdateStatsError(errors int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Errors += errors
}

// UpdateStatsError increment the skipped items
func (s *Stats) UpdateStatsSkipped(skipped int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Skipped += skipped
}
