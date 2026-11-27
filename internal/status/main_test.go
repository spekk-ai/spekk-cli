package status

import (
	"os"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/testenv"
)

// TestMain isolates the tests from the developer's git config and keeps them
// off the network.
func TestMain(m *testing.M) {
	os.Exit(testenv.Main(m))
}
