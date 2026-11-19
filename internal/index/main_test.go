package index

import (
	"os"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/testenv"
)

// TestMain isolates the tests from the developer's git config.
func TestMain(m *testing.M) {
	os.Exit(testenv.Main(m))
}
