package dbkit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestHealth_IsHealthy(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test healthy database
	if !db.IsHealthy(ctx) {
		t.Error("Database should be healthy")
	}
}

func TestHealth_Health(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test detailed health status
	status := db.Health(ctx)

	if !status.Healthy {
		t.Error("Database should be healthy")
	}

	if status.Latency <= 0 {
		t.Error("Latency should be positive")
	}

	if status.PoolStats.InUse < 0 {
		t.Error("InUse connections should not be negative")
	}

	if status.PoolStats.Idle < 0 {
		t.Error("Idle connections should not be negative")
	}

	if status.PoolStats.MaxOpenConnections <= 0 {
		t.Error("MaxOpenConnections should be positive")
	}

	// PoolStats should have reasonable values
	if status.PoolStats.MaxOpenConnections > 1000 {
		t.Errorf("MaxOpenConnections seems too high: %d", status.PoolStats.MaxOpenConnections)
	}
}

func TestHealth_ConnectionPoolStats(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Perform multiple operations to exercise connection pool
	for i := 0; i < 5; i++ {
		model := &TestModel{
			Name:  "Health Test",
			Email: fmt.Sprintf("health%d@example.com", i),
			Age:   25 + i,
		}
		_, err = db.NewInsert().Model(model).Exec(ctx)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Check stats after operations
	stats := db.Stats()

	if stats.OpenConnections <= 0 {
		t.Error("Should have open connections")
	}

	if stats.InUse < 0 {
		t.Error("InUse should not be negative")
	}

	if stats.Idle < 0 {
		t.Error("Idle should not be negative")
	}

	if stats.MaxOpenConnections <= 0 {
		t.Error("MaxOpenConnections should be positive")
	}

	// Check health status includes pool stats
	status := db.Health(ctx)

	if status.PoolStats.OpenConnections != stats.OpenConnections {
		t.Errorf("Pool stats mismatch: expected %d, got %d",
			stats.OpenConnections, status.PoolStats.OpenConnections)
	}

	if status.PoolStats.InUse != stats.InUse {
		t.Errorf("Pool InUse mismatch: expected %d, got %d",
			stats.InUse, status.PoolStats.InUse)
	}

	if status.PoolStats.Idle != stats.Idle {
		t.Errorf("Pool Idle mismatch: expected %d, got %d",
			stats.Idle, status.PoolStats.Idle)
	}
}

func TestHealth_MultipleHealthChecks(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Perform multiple health checks
	for i := 0; i < 3; i++ {
		status := db.Health(ctx)

		if !status.Healthy {
			t.Errorf("Health check %d failed", i)
		}

		if status.Latency <= 0 {
			t.Errorf("Latency should be positive on check %d", i)
		}

		// Small delay between checks
		time.Sleep(10 * time.Millisecond)
	}
}

func TestHealth_LatencyMeasurement(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test latency measurement
	start := time.Now()
	status := db.Health(ctx)
	duration := time.Since(start)

	if !status.Healthy {
		t.Error("Database should be healthy")
	}

	// Latency should be reasonable (less than 1 second for local test)
	if status.Latency > time.Second {
		t.Errorf("Latency seems too high: %v", status.Latency)
	}

	// Health check should complete quickly
	if duration > 5*time.Second {
		t.Errorf("Health check took too long: %v", duration)
	}

	// Latency should be less than total duration
	if status.Latency > duration {
		t.Errorf("Latency (%v) should not exceed total duration (%v)",
			status.Latency, duration)
	}
}

func TestHealth_Ping(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test direct ping
	err := db.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Test multiple pings
	for i := 0; i < 5; i++ {
		err := db.Ping(ctx)
		if err != nil {
			t.Errorf("Ping %d failed: %v", i, err)
		}
	}
}

func TestHealth_ContextCancellation(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Health check should complete before timeout
	status := db.Health(ctx)

	if !status.Healthy {
		t.Error("Database should be healthy")
	}

	if status.Latency > 100*time.Millisecond {
		t.Errorf("Latency (%v) should be less than context timeout (100ms)", status.Latency)
	}
}

func TestHealth_ContextCancellation_Ping(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Ping should complete before timeout
	err := db.Ping(ctx)
	if err != nil {
		t.Errorf("Ping with context failed: %v", err)
	}
}

func TestHealth_StatsConsistency(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Get stats multiple times and check consistency
	stats1 := db.Stats()
	status1 := db.Health(ctx)

	stats2 := db.Stats()
	status2 := db.Health(ctx)

	// Stats should be consistent within reason
	if stats1.MaxOpenConnections != stats2.MaxOpenConnections {
		t.Error("MaxOpenConnections should be consistent")
	}

	// Health status pool stats should match direct stats
	if status1.PoolStats.MaxOpenConnections != stats1.MaxOpenConnections {
		t.Error("Health pool stats should match direct stats")
	}

	if status2.PoolStats.MaxOpenConnections != stats2.MaxOpenConnections {
		t.Error("Health pool stats should match direct stats")
	}
}

func TestHealth_WithLoad(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := createTable(t, db)

	// Perform concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 5; j++ {
				model := &TestModel{
					Name:  "Load Test",
					Email: fmt.Sprintf("load_%d_%d@example.com", id, j),
					Age:   id,
				}
				_, err := db.NewInsert().Model(model).Exec(ctx)
				if err != nil {
					t.Errorf("Create failed in goroutine %d: %v", id, err)
					return
				}
			}
		}(i)
	}

	// Perform health checks while load is running
	for i := 0; i < 5; i++ {
		status := db.Health(ctx)
		if !status.Healthy {
			t.Errorf("Health check %d failed under load", i)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final health check
	finalStatus := db.Health(ctx)
	if !finalStatus.Healthy {
		t.Error("Final health check failed")
	}

	// Check that pool stats are reasonable
	if finalStatus.PoolStats.InUse < 0 {
		t.Error("InUse should not be negative after load")
	}

	if finalStatus.PoolStats.Idle < 0 {
		t.Error("Idle should not be negative after load")
	}
}

func createTable(t *testing.T, db *DBKit) context.Context {
	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Clean up existing data
	_, _ = db.NewDelete().Model((*TestModel)(nil)).Where("1=1").Exec(ctx)
	return ctx
}
