package migrations

import "testing"

func TestUpRequiresIntegrationEnvironment(t *testing.T) {
	t.Skip("integration test requires a live PostgreSQL pool")
}
