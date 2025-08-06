//go:build integration

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nudgebot-api/internal/nudge"
)

func TestMigration_RunMigrationsFailure(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Test migration failure by attempting to migrate with corrupted database connection
	// Close the database connection to simulate failure
	sqlDB, err := testDB.DB.DB()
	require.NoError(t, err)
	sqlDB.Close()

	// Attempt to run migrations on closed database
	err = nudge.RunMigrations(testDB.DB)
	assert.Error(t, err, "Migration should fail with closed database connection")
	assert.Contains(t, err.Error(), "failed to auto-migrate")
}

func TestMigration_ValidateMigrationsFailure(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Don't run migrations, so validation should fail
	err := nudge.ValidateMigrations(testDB.DB)
	assert.Error(t, err, "Validation should fail when tables don't exist")
	assert.Contains(t, err.Error(), "table")
}

func TestMigration_MigrateWithValidationFailure(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Test the complete migration flow with a database that will fail validation
	// First close the connection to simulate failure
	sqlDB, err := testDB.DB.DB()
	require.NoError(t, err)
	sqlDB.Close()

	// Attempt complete migration with validation
	err = nudge.MigrateWithValidation(testDB.DB)
	assert.Error(t, err, "Migration with validation should fail")
}

func TestMigration_PartialMigrationRecovery(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Create only some tables manually to simulate partial migration
	err := testDB.DB.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			description TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	// Attempt to complete the migration
	err = nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Verify all tables exist after recovery
	var tables []string
	err = testDB.DB.Raw(`
		SELECT tablename FROM pg_tables 
		WHERE schemaname = 'public' 
		AND tablename IN ('tasks', 'reminders', 'nudge_settings')
	`).Scan(&tables).Error
	require.NoError(t, err)

	// Should have all expected tables
	assert.Contains(t, tables, "tasks")
	assert.Contains(t, tables, "reminders")
	assert.Contains(t, tables, "nudge_settings")
}

func TestMigration_DatabaseConnectionFailure(t *testing.T) {
	// Test migration behavior when database becomes unavailable
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Close database connection to simulate failure
	sqlDB, err := testDB.DB.DB()
	require.NoError(t, err)
	sqlDB.Close()

	// Test various migration operations with failed connection
	err = nudge.RunMigrations(testDB.DB)
	assert.Error(t, err, "Should fail when database is unavailable")

	err = nudge.ValidateMigrations(testDB.DB)
	assert.Error(t, err, "Should fail when database is unavailable")

	err = nudge.MigrateWithValidation(testDB.DB)
	assert.Error(t, err, "Should fail when database is unavailable")
}

func TestMigration_MigrationIdempotency(t *testing.T) {
	// Test that running migrations multiple times is safe
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Run migrations first time
	err := nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Run migrations second time - should be idempotent
	err = nudge.RunMigrations(testDB.DB)
	assert.NoError(t, err, "Migrations should be idempotent")

	// Validation should pass after successful migrations
	err = nudge.ValidateMigrations(testDB.DB)
	assert.NoError(t, err)

	// Run complete migration with validation multiple times
	err = nudge.MigrateWithValidation(testDB.DB)
	assert.NoError(t, err, "Migration with validation should be idempotent")
}

func TestMigration_TableStructureValidation(t *testing.T) {
	// Test that migrations create the expected table structure
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Run migrations
	err := nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Verify tasks table structure
	var columns []string
	err = testDB.DB.Raw(`
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = 'tasks' 
		AND table_schema = 'public'
		ORDER BY ordinal_position
	`).Scan(&columns).Error
	require.NoError(t, err)

	// Check for expected columns
	expectedColumns := []string{"id", "user_id", "description"}
	for _, expectedCol := range expectedColumns {
		assert.Contains(t, columns, expectedCol, "Tasks table should have %s column", expectedCol)
	}

	// Verify reminders table exists
	var reminderColumns []string
	err = testDB.DB.Raw(`
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = 'reminders' 
		AND table_schema = 'public'
	`).Scan(&reminderColumns).Error
	require.NoError(t, err)
	assert.NotEmpty(t, reminderColumns, "Reminders table should have columns")

	// Verify nudge_settings table exists
	var settingsColumns []string
	err = testDB.DB.Raw(`
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = 'nudge_settings' 
		AND table_schema = 'public'
	`).Scan(&settingsColumns).Error
	require.NoError(t, err)
	assert.NotEmpty(t, settingsColumns, "Nudge settings table should have columns")
}

func TestMigration_ConstraintValidation(t *testing.T) {
	// Test that database constraints are properly created
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Run migrations
	err := nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Test primary key constraints by attempting to insert duplicate IDs
	task1 := nudge.Task{
		ID:          "test-task-1",
		UserID:      "user1",
		Description: "Test task 1",
		Status:      "active",
		Priority:    "medium",
	}

	task2 := nudge.Task{
		ID:          "test-task-1", // Same ID
		UserID:      "user2",
		Description: "Test task 2",
		Status:      "active",
		Priority:    "high",
	}

	// First insert should succeed
	err = testDB.DB.Create(&task1).Error
	assert.NoError(t, err)

	// Second insert with same ID should fail due to primary key constraint
	err = testDB.DB.Create(&task2).Error
	assert.Error(t, err, "Should fail due to primary key constraint")
	assert.Contains(t, err.Error(), "duplicate")
}

func TestMigration_IndexValidation(t *testing.T) {
	// Test that indexes are created correctly
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Run migrations with indexes
	err := nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Verify indexes exist using the ValidateMigrations function
	err = nudge.ValidateMigrations(testDB.DB)
	assert.NoError(t, err, "Validation should pass when all indexes are created")

	// Check specific indexes exist
	var indexes []string
	err = testDB.DB.Raw(`
		SELECT indexname 
		FROM pg_indexes 
		WHERE tablename = 'tasks' 
		AND schemaname = 'public'
		AND indexname LIKE 'idx_%'
	`).Scan(&indexes).Error
	require.NoError(t, err)

	// Should have multiple indexes on tasks table
	assert.NotEmpty(t, indexes, "Should have indexes on tasks table")

	// Look for user_id index specifically
	hasUserIDIndex := false
	for _, index := range indexes {
		if index == "idx_tasks_user_id" {
			hasUserIDIndex = true
			break
		}
	}
	assert.True(t, hasUserIDIndex, "Should have index on user_id column")
}

func TestMigration_TableStats(t *testing.T) {
	// Test the GetTableStats function works correctly
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Run migrations
	err := nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Get initial stats (should be empty)
	stats, err := nudge.GetTableStats(testDB.DB)
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats["tasks"])
	assert.Equal(t, int64(0), stats["reminders"])
	assert.Equal(t, int64(0), stats["nudge_settings"])

	// Create a test task
	task := nudge.Task{
		ID:          "test-task-stats",
		UserID:      "user1",
		Description: "Test task for stats",
		Status:      "active",
		Priority:    "medium",
	}

	err = testDB.DB.Create(&task).Error
	require.NoError(t, err)

	// Get updated stats
	stats, err = nudge.GetTableStats(testDB.DB)
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats["tasks"])
	assert.Equal(t, int64(0), stats["reminders"])
	assert.Equal(t, int64(0), stats["nudge_settings"])
}

func TestMigration_DropTablesRecovery(t *testing.T) {
	// Test that DropTables works and migrations can recover
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Run initial migrations
	err := nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Drop all tables
	err = nudge.DropTables(testDB.DB)
	require.NoError(t, err)

	// Validation should fail after dropping tables
	err = nudge.ValidateMigrations(testDB.DB)
	assert.Error(t, err, "Validation should fail after dropping tables")

	// Re-run migrations to recover
	err = nudge.RunMigrations(testDB.DB)
	require.NoError(t, err)

	// Validation should pass after recovery
	err = nudge.ValidateMigrations(testDB.DB)
	assert.NoError(t, err, "Validation should pass after recovery")
}
