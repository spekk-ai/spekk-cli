package sandbox

import (
	"fmt"
	"strings"
)

const codexVersion = "0.163.0"
const codexHealthCommand = `set -e
case "$(dpkg --print-architecture)" in
  amd64) host_sha=8724d73ec190536233ab4628d24a071bfbcf27e5124f08fe35129f26fe48b916 ;;
  arm64) host_sha=255eccc8a5c1d267dc0acbc815cc83aaa1f50955bd99943b29f00cafcdcab771 ;;
  *) echo 'Codex requires amd64 or arm64' >&2; exit 1 ;;
esac
echo "$host_sha  /usr/local/bin/codex-code-mode-host" | sha256sum -c - >/dev/null
/usr/local/bin/codex-code-mode-host --help >/dev/null
/usr/local/bin/codex --version`

// The GitHub release contains one binary per Linux architecture. Download to
// a temporary path, then replace the installed binary only after it succeeds.
const codexDownloadScript = `set -e
case "$(dpkg --print-architecture)" in
  amd64) arch=x86_64 ;;
  arm64) arch=aarch64 ;;
  *) echo 'Codex requires amd64 or arm64' >&2; exit 1 ;;
esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "https://github.com/openai/codex/releases/download/rust-v0.163.0/codex-${arch}-unknown-linux-musl.tar.gz" -o "$tmp/codex.tar.gz"
curl -fsSL "https://github.com/openai/codex/releases/download/rust-v0.163.0/codex-code-mode-host-${arch}-unknown-linux-musl.tar.gz" -o "$tmp/codex-code-mode-host.tar.gz"
tar -xzf "$tmp/codex.tar.gz" -C "$tmp"
tar -xzf "$tmp/codex-code-mode-host.tar.gz" -C "$tmp"
install -m 0755 "$tmp/codex-${arch}-unknown-linux-musl" /usr/local/bin/codex
install -m 0755 "$tmp/codex-code-mode-host-${arch}-unknown-linux-musl" /usr/local/bin/codex-code-mode-host`

const codexInstallScript = codexDownloadScript + "\n" + codexHealthCommand

func ensureCodexCLI(ip, keyPath, name, user string) error {
	out, err := runSSHCombined(ip, keyPath, name, user, privilegedScript(user, codexHealthCommand))
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
