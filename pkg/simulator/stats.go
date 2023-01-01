// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package simulator contains the core simulation coordinator
package simulator

import (
	"github.com/onosproject/onos-api/go/onos/misc"
	"sync"
	"time"
)

// StatsCollector drives collection of I/O statistics as a time-series of rates and counters
type StatsCollector struct {
	simulation *Simulation
	lock       sync.RWMutex
	stats      []*misc.IOStats
	lastTime   time.Time
}

func newStatsCollector(sim *Simulation) *StatsCollector {
	return &StatsCollector{
		simulation: sim,
		stats:      make([]*misc.IOStats, 0, 1000),
		lastTime:   time.Now(),
	}
}

// Start starts the I/O stats collector in the background
func (c *StatsCollector) Start() {
	go c.collect()
}

// GetIOStats returns the list of I/O stats.
func (c *StatsCollector) GetIOStats() []*misc.IOStats {
	c.lock.RLock()
	defer c.lock.RUnlock()
	stats := c.stats
	return stats
}

func (c *StatsCollector) collect() {
	for {
		time.Sleep(5 * time.Second)
		c.createSample()
	}
}

func (c *StatsCollector) createSample() {
	c.lock.Lock()
	defer c.lock.Unlock()
	now := time.Now()
	stats := &misc.IOStats{
		FirstUpdateTime: uint64(c.lastTime.UnixNano()),
		LastUpdateTime:  uint64(now.UnixNano()),
	}
	for _, dsim := range c.simulation.deviceSimulators {
		dsim.addAndResetStats(stats.LastUpdateTime, stats)
	}
	if len(c.stats) < cap(c.stats) {
		c.stats = append(c.stats, stats)
	} else {
		c.stats = append(c.stats[1:], stats)
	}
	c.lastTime = now
}
