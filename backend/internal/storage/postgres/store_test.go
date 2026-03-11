package postgres

import "testing"

// TestRebindQuery verifies that PostgreSQL placeholder rewriting keeps the SQL
// text stable except for sequentially numbered parameters.
func TestRebindQuery(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain placeholders",
			input:    "SELECT * FROM users WHERE id = ? AND username = ? AND trust_level >= ?",
			expected: "SELECT * FROM users WHERE id = $1 AND username = $2 AND trust_level >= $3",
		},
		{
			name:     "question marks inside string literals stay untouched",
			input:    "SELECT '?literal?' AS marker, id FROM users WHERE username = ?",
			expected: "SELECT '?literal?' AS marker, id FROM users WHERE username = $1",
		},
		{
			name:     "question marks inside quoted identifiers stay untouched",
			input:    "SELECT \"weird?column\" FROM users WHERE id = ?",
			expected: "SELECT \"weird?column\" FROM users WHERE id = $1",
		},
		{
			name:     "question marks inside line comments stay untouched",
			input:    "SELECT id -- keep ? comment\nFROM users WHERE username = ?",
			expected: "SELECT id -- keep ? comment\nFROM users WHERE username = $1",
		},
		{
			name:     "question marks inside block comments stay untouched",
			input:    "SELECT /* keep ? comment */ id FROM users WHERE username = ? AND trust_level >= ?",
			expected: "SELECT /* keep ? comment */ id FROM users WHERE username = $1 AND trust_level >= $2",
		},
		{
			name:     "escaped quotes do not break state tracking",
			input:    "SELECT 'it''s ? literal' AS marker FROM users WHERE username = ?",
			expected: "SELECT 'it''s ? literal' AS marker FROM users WHERE username = $1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if actual := rebindQuery(testCase.input); actual != testCase.expected {
				t.Fatalf("expected %q, got %q", testCase.expected, actual)
			}
		})
	}
}
