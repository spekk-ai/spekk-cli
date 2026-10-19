package sandbox

import (
	"fmt"
	"strings"
)

const codexVersion = "0.163.0"

// The GitHub release contains one binary per Linux architecture. Download to
// a temporary path, then replace the installed binary only after it succeeds.
const codexInstallScript = `set -e
case "$(dpkg --print-architecture)" in
  amd64) arch=x86_64 ;;
  arm64) arch=aarch64 ;;
  *) echo 'Codex requires amd64 or arm64' >&2; exit 1 ;;
esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "https://github.com/openai/codex/releases/download/rust-v0.163.0/codex-${arch}-unknown-linux-musl.tar.gz" -o "$tmp/codex.tar.gz"
tar -xzf "$tmp/codex.tar.gz" -C "$tmp"
install -m 0755 "$tmp/codex-${arch}-unknown-linux-musl" /usr/local/bin/codex
/usr/local/bin/codex --version`

func ensureCodexCLI(ip, keyPath, name, user string) error {
	out, err := runSSHCombined(ip, keyPath, name, user, privilegedScript(user, "/usr/local/bin/codex --version"))
	if err == nil && strings.TrimSpace(out) == "codex-cli "+codexVersion {
		return nil
	}
	out, err = runSSHCombined(ip, keyPath, name, user, privilegedScript(user, codexInstallScript))
	if err != nil {
		return fmt.Errorf("installing Codex CLI: %s\n%s", err, out)
	}
	if strings.TrimSpace(out) != "codex-cli "+codexVersion {
		return fmt.Errorf("Codex CLI version check failed: %s", out)
	}
	return nil
}
