package logger

import (
	"fmt"
	"os"
	"testing"
)

func TestEnvVarSet(t *testing.T) {
	varName := "TEST_VAR"

	testRows := []struct {
		v        string
		expected bool
	}{
		{"1", true},
		{"yes", true},
		{"not_set", false},
		{"no", false},
		{"false", false},
	}

	for _, row := range testRows {
		t.Run(fmt.Sprintf("Val: %s, should be set: %v", row.v, row.expected), func(t *testing.T) {
			os.Unsetenv(varName)
			if row.v != "not_set" {
				os.Setenv(varName, row.v)
			}

			got := envVarSet(varName)
			if got != row.expected {
				t.Errorf("Variable set: got=%v, expected=%v", got, row.expected)
			}
		})
	}
}
