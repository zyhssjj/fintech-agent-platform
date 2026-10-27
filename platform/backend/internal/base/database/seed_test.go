package database

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestMigrationAdminPasswordHashMatchesDefaultPassword(t *testing.T) {
	const migrationAdminPasswordHash = "$2a$10$tOdipUFcODKLMZwKnoCMyucPPKi4I7TftT0DWc3YgV7jHsB6YPx3u"

	if err := bcrypt.CompareHashAndPassword([]byte(migrationAdminPasswordHash), []byte(DefaultAdminPassword)); err != nil {
		t.Fatalf("migration admin password hash does not match default password: %v", err)
	}
}

func TestShouldRepairLegacyBrokenAdminPasswordHash(t *testing.T) {
	const legacyBrokenHash = "$2a$10$CUGqJ5ksLKdbw18nGX7one88/ApRlKVmoi6JrnZSM8ciJ2WF30Sfy"

	if !shouldRepairAdminPassword(legacyBrokenHash) {
		t.Fatal("legacy broken admin password hash should be repaired")
	}
}

func TestShouldNotRepairCustomAdminPasswordHash(t *testing.T) {
	customHash, err := bcrypt.GenerateFromPassword([]byte("custom-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	if shouldRepairAdminPassword(string(customHash)) {
		t.Fatal("custom admin password hash should not be repaired")
	}
}
