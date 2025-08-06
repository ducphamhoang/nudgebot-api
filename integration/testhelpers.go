//go:build integration

package integration

import (
	"testing"

	main "nudgebot-api"

	"gorm.io/gorm"
)

// SetupCentralizedTestDatabase is a wrapper around the centralized test database setup function
// This demonstrates how to use the centralized setup instead of defining a custom one
func SetupCentralizedTestDatabase(t *testing.T) (*gorm.DB, func()) {
	// Use the centralized setup function from the main package
	testContainer := main.SetupTestDatabase(t)
	
	// Return the database and a cleanup function that matches the expected signature
	cleanup := func() {
		testContainer.TeardownTestDatabase(t)
	}
	
	return testContainer.DB, cleanup
}
