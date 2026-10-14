package postgres_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"passwordmanage/server/internal/adapter/postgres"
	"passwordmanage/server/internal/testdb"
)

func TestPoolCollectorReportsLiveStats(t *testing.T) {
	pool := testdb.New(t)
	reg := prometheus.NewRegistry()
	reg.MustRegister(postgres.NewPoolCollector(pool))

	// Pedantic lint catches mismatches between Describe and Collect.
	problems, err := testutil.CollectAndLint(postgres.NewPoolCollector(pool))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("metric lint problems: %+v", problems)
	}

	want := `
# HELP pm_db_pool_max_conns Maximum connections the pool may open.
# TYPE pm_db_pool_max_conns gauge
pm_db_pool_max_conns ` + strconv.Itoa(int(pool.Config().MaxConns)) + "\n"
	if err := testutil.GatherAndCompare(reg, strings.NewReader(want), "pm_db_pool_max_conns"); err != nil {
		t.Fatal(err)
	}
}
