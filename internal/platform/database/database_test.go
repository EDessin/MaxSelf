package database

import "testing"

func TestOpenReturnsErrorAfterRetryLimit(t *testing.T) {
	originalAttempts := openAttempts
	originalSleep := openSleep
	openAttempts = 1
	openSleep = 0
	t.Cleanup(func() {
		openAttempts = originalAttempts
		openSleep = originalSleep
	})

	db, err := Open("://not-a-valid-dsn")
	if err == nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		t.Fatal("expected invalid database URL to fail")
	}
}
