package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolCollector exports connection-pool statistics, read fresh on every scrape.
type PoolCollector struct {
	pool *pgxpool.Pool

	maxConns, totalConns, idleConns, acquiredConns *prometheus.Desc
	acquireCount, emptyAcquireCount, canceledCount *prometheus.Desc
	acquireSeconds, emptyAcquireWaitSeconds        *prometheus.Desc
}

func NewPoolCollector(pool *pgxpool.Pool) *PoolCollector {
	desc := func(name, help string) *prometheus.Desc {
		return prometheus.NewDesc("pm_db_pool_"+name, help, nil, nil)
	}
	return &PoolCollector{
		pool:                    pool,
		maxConns:                desc("max_conns", "Maximum connections the pool may open."),
		totalConns:              desc("total_conns", "Connections currently open."),
		idleConns:               desc("idle_conns", "Open connections not in use."),
		acquiredConns:           desc("acquired_conns", "Connections currently in use."),
		acquireCount:            desc("acquires_total", "Successful connection acquisitions."),
		emptyAcquireCount:       desc("empty_acquires_total", "Acquisitions that had to wait because no idle connection was free."),
		canceledCount:           desc("canceled_acquires_total", "Acquisitions canceled by their context."),
		acquireSeconds:          desc("acquire_seconds_total", "Total time spent acquiring connections."),
		emptyAcquireWaitSeconds: desc("empty_acquire_wait_seconds_total", "Total time spent waiting for a free connection."),
	}
}

func (c *PoolCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{
		c.maxConns, c.totalConns, c.idleConns, c.acquiredConns,
		c.acquireCount, c.emptyAcquireCount, c.canceledCount,
		c.acquireSeconds, c.emptyAcquireWaitSeconds,
	} {
		ch <- d
	}
}

func (c *PoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	gauge := func(d *prometheus.Desc, v float64) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v)
	}
	counter := func(d *prometheus.Desc, v float64) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.CounterValue, v)
	}
	gauge(c.maxConns, float64(s.MaxConns()))
	gauge(c.totalConns, float64(s.TotalConns()))
	gauge(c.idleConns, float64(s.IdleConns()))
	gauge(c.acquiredConns, float64(s.AcquiredConns()))
	counter(c.acquireCount, float64(s.AcquireCount()))
	counter(c.emptyAcquireCount, float64(s.EmptyAcquireCount()))
	counter(c.canceledCount, float64(s.CanceledAcquireCount()))
	counter(c.acquireSeconds, s.AcquireDuration().Seconds())
	counter(c.emptyAcquireWaitSeconds, s.EmptyAcquireWaitTime().Seconds())
}
