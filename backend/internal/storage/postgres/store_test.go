package postgres

import "testing"

// TestRebindQuery verifies that PostgreSQL placeholder rewriting keeps the SQL
// text stable except for sequentially numbered parameters.
func TestRebindQuery(t *testing.T) {
	input := "SELECT * FROM users WHERE id = ? AND username = ? AND trust_level >= ?"
	expected := "SELECT * FROM users WHERE id = $1 AND username = $2 AND trust_level >= $3"

	if actual := rebindQuery(input); actual != expected {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
