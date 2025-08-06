//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"nudgebot-api/internal/config"
	"nudgebot-api/internal/database"
)

// TestContainer wraps database container for testing
type TestContainer struct {
	Container testcontainers.Container
	DB        *gorm.DB
	Config    config.DatabaseConfig
	ctx       context.Context
}

// SetupTestDatabase creates a test database using testcontainers
func SetupTestDatabase(t *testing.T) (*TestContainer, func()) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test_nudgebot"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Configure database connection
	dbConfig := config.DatabaseConfig{
		Host:            host,
		Port:            port.Int(),
		User:            "test_user",
		Password:        "test_password",
		DBName:          "test_nudgebot",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	// Connect to database
	db, err := database.NewPostgresConnection(dbConfig)
	require.NoError(t, err, "Failed to connect to test database")

	testContainer := &TestContainer{
		Container: postgresContainer,
		DB:        db,
		Config:    dbConfig,
		ctx:       ctx,
	}

	cleanup := func() {
		if db != nil {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate postgres container: %s", err)
		}
	}

	return testContainer, cleanup
}

// SetupCentralizedTestDatabase is a wrapper around the centralized test database setup function
// This demonstrates how to use the centralized setup instead of defining a custom one
func SetupCentralizedTestDatabase(t *testing.T) (*gorm.DB, func()) {
	testContainer, cleanup := SetupTestDatabase(t)

	// Return the database and a cleanup function that matches the expected signature
	return testContainer.DB, cleanup
}
