package appclient

import (
	"os"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/testenv"
)

// TestMain fails the run when a test changed the developer's real spekk
// config directory.
func TestMain(m *testing.M) {
	os.Exit(testenv.Main(m))
}
