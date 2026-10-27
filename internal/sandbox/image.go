package sandbox

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/config"
	"github.com/spekk-ai/spekk-cli/internal/version"
)

// Template images: `spekk sandbox image build` makes a provider image from a
// builder sandbox, and `spekk sandbox create --image` starts a sandbox from
// it. See specs/sandbox-template-images/.

// imageCloudInit is the first-boot cloud-init of a sandbox made from an
// image. The packages are in the image, so it only gives agent the sandbox
// key and writes the marker.
//
//go:embed cloud-init-image.yaml
var imageCloudInit []byte

// imageMaxAge is the age after which create warns that an image needs a
// rebuild for security updates.
const imageMaxAge = 14 * 24 * time.Hour

// ErrImageNotFound reports that the provider has no image with that ID.
var ErrImageNotFound = errors.New("image not found")

// Imager is a Provider that can make images of its machines and create
// machines from them. It is optional: create falls back to the full
// cloud-init for a provider without it.
type Imager interface {
	// CreateImage stops the machine of meta and makes a private image of
	// it named name. A provider whose images belong to one region copies
	// the image to each of regions. On an error after the image exists,
	// it returns the image as well, so the caller can record it and
	// delete it later.
	CreateImage(meta *SandboxMeta, name string, regions []string) (ImageMeta, error)
	// DeleteImage deletes the provider image. It returns an error that
	// wraps ErrImageNotFound when the image is not there.
	DeleteImage(img ImageMeta) error
	// ListImages lists the private images of the account, with the name,
	// ID, regions, and created date that the provider knows.
	ListImages() ([]ImageMeta, error)
	// ImageRegion is the region that a create with opts uses, which an
	// image must have a copy in. Empty means the images are global.
	ImageRegion(opts CreateOptions) string
}

// ImageMeta is the local record of an image, in images.json.
type ImageMeta struct {
	// Name is the key of the record in images.json.
	Name     string `json:"-"`
	Provider string `json:"provider"`
	// ImageID is the provider's ID: a snapshot ID on DigitalOcean, the
	// image name on GCP.
	ImageID string `json:"imageId"`
	// Regions hold a copy of the image. Empty means the image is global.
	Regions []string `json:"regions,omitempty"`
	// Project is the GCP project of the image.
	Project      string `json:"project,omitempty"`
	CreatedAt    string `json:"createdAt"`
	SpekkVersion string `json:"spekkVersion,omitempty"`
	OSVersion    string `json:"osVersion,omitempty"`
	// SetupSHA256 is the hash of the --setup script; empty with none.
	SetupSHA256 string `json:"setupSha256,omitempty"`
}

// age is how long ago the image was made, or zero when that is not known.
func (img ImageMeta) age(now time.Time) time.Duration {
	t, err := time.Parse(time.RFC3339, img.CreatedAt)
	if err != nil {
		return 0
	}
	return now.Sub(t)
}

// --- Store ---

// imagesFile returns the path of images.json. Overridable in tests.
var imagesFile = func() string {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		dir = config.DefaultDir()
	}
	return filepath.Join(dir, "images.json")
}

// LoadImages reads the image records.
func LoadImages() (map[string]*ImageMeta, error) {
	images := map[string]*ImageMeta{}
	data, err := os.ReadFile(imagesFile())
	if errors.Is(err, os.ErrNotExist) {
		return images, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &images); err != nil {
		return nil, fmt.Errorf("parsing images.json: %w", err)
	}
	if images == nil {
		images = map[string]*ImageMeta{}
	}
	for name, img := range images {
		img.Name = name
	}
	return images, nil
}

// saveImages writes the image records.
func saveImages(images map[string]*ImageMeta) error {
	if err := os.MkdirAll(filepath.Dir(imagesFile()), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(images, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(imagesFile(), append(data, '\n'), 0o600)
}

// SaveImage writes or replaces one image record.
func SaveImage(img ImageMeta) error {
	images, err := LoadImages()
	if err != nil {
		return err
	}
	images[img.Name] = &img
	return saveImages(images)
}

// --- Build ---

var imageNameRe = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateImageName checks a name that both DigitalOcean and GCP accept.
func ValidateImageName(name string) error {
	if !imageNameRe.MatchString(name) {
		return fmt.Errorf("invalid image name %q: must match [a-z]([a-z0-9-]{0,61}[a-z0-9])? (lowercase, starting with a letter, at most 63 characters)", name)
	}
	return nil
}

// ImageBuildOptions holds the flags of `spekk sandbox image build`.
type ImageBuildOptions struct {
	Name    string
	Setup   string // path of the setup script; empty for the base image
	Size    string
	Regions []string // DigitalOcean: the builder region, then the copies
	Project string
}

// DefaultImageRegions are the DigitalOcean regions an image is copied to
// when --regions does not say.
var DefaultImageRegions = []string{"nyc1", "nyc3"}

// builderName is the sandbox name of the builder of an image.
func builderName(image string) string { return "image-" + image }

// BuildImage makes a provider image: a builder sandbox with the normal
// cloud-init, the optional setup script, the hygiene step, then the image.
// The builder is destroyed on success, on each failure, and on Ctrl-C.
func BuildImage(p Provider, opts ImageBuildOptions) (err error) {
	if err := ValidateImageName(opts.Name); err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("an image needs a cloud provider: spekk does not make images of a machine it did not create")
	}
	imager, ok := p.(Imager)
	if !ok {
		return fmt.Errorf("provider %s cannot make images", p.Name())
	}
	images, err := LoadImages()
	if err != nil {
		return err
	}
	if _, ok := images[opts.Name]; ok {
		return fmt.Errorf("image %q already exists; delete it first: spekk sandbox image delete %s", opts.Name, opts.Name)
	}
	var setup []byte
	if opts.Setup != "" {
		if setup, err = os.ReadFile(opts.Setup); err != nil {
			return fmt.Errorf("reading --setup: %w", err)
		}
	}
	name := builderName(opts.Name)
	if existing, err := GetSandbox(name); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("builder sandbox %q already exists (%s); destroy it first: spekk sandbox destroy %s --force", name, machineRef(existing), name)
	}

	meta := &SandboxMeta{
		Provider:  p.Name(),
		Status:    "building",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Kind:      KindOneOff,
		Purpose:   "image builder for " + opts.Name,
		CreatedBy: hostname(),
	}
	var once sync.Once
	// Ctrl-C during a build must not leave a builder that bills.
	cleanup := func() { once.Do(func() { destroyBuilder(p, name, meta) }) }
	stop := cleanupOnSignal(cleanup)
	defer func() {
		stop()
		cleanup()
	}()

	// A known_hosts file from an earlier builder of this name holds its old
	// host key, and each probe of the new builder would fail on it.
	if err := os.Remove(KnownHostsFile(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing the old host key of %s: %w", name, err)
	}
	createOpts := CreateOptions{Name: name, Size: opts.Size, Project: opts.Project, CloudInit: cloudInitTemplate}
	if len(opts.Regions) > 0 {
		createOpts.Region = opts.Regions[0]
	}
	fmt.Fprintf(os.Stderr, "Creating builder sandbox %s...\n", name)
	createErr := p.Create(name, createOpts, meta)
	if namesMachine(meta) {
		if err := SaveSandbox(name, meta); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not record %s: %s\n", machineRef(meta), err)
		}
	}
	if createErr != nil {
		return fmt.Errorf("creating the builder: %w", createErr)
	}
	if err := waitReady(meta.IP, meta.SSHKeyPath, name, sshUser(meta), DefaultProvisionTimeout); err != nil {
		return fmt.Errorf("waiting for the builder: %w", err)
	}

	if setup != nil {
		fmt.Fprintf(os.Stderr, "Running %s on the builder as root...\n", opts.Setup)
		if err := runSetup(meta, name, string(setup)); err != nil {
			return fmt.Errorf("running %s: %w", opts.Setup, err)
		}
	}

	fmt.Fprintln(os.Stderr, "Cleaning the builder and checking it for credentials...")
	osVersion, err := checkHygiene(meta, name)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Making image %q...\n", opts.Name)
	img, imgErr := imager.CreateImage(meta, opts.Name, opts.Regions)
	if img.ImageID == "" {
		if imgErr == nil {
			imgErr = errors.New("the provider returned no image ID")
		}
		return fmt.Errorf("making the image: %w", imgErr)
	}
	img.Name = opts.Name
	img.Provider = p.Name()
	img.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	img.SpekkVersion = version.Version
	img.OSVersion = osVersion
	if setup != nil {
		sum := sha256.Sum256(setup)
		img.SetupSHA256 = hex.EncodeToString(sum[:])
	}
	if err := SaveImage(img); err != nil {
		return errors.Join(imgErr, fmt.Errorf("recording image %s (%s): %w", img.Name, img.ImageID, err))
	}
	if imgErr != nil {
		return fmt.Errorf("making the image: %w\nImage %s (%s) exists in %s and is recorded. Delete it with: spekk sandbox image delete %s",
			imgErr, img.Name, img.ImageID, regionsOrGlobal(img), img.Name)
	}
	fmt.Fprintf(os.Stderr, "Image %s (%s) is ready in %s.\n", img.Name, img.ImageID, regionsOrGlobal(img))
	return nil
}

// regionsOrGlobal names where an image is, for a message.
func regionsOrGlobal(img ImageMeta) string {
	if len(img.Regions) == 0 {
		return "all regions"
	}
	return strings.Join(img.Regions, ", ")
}

// destroyBuilder destroys the builder machine, its record, its key pair, and
// its known_hosts file, and says what it removed. A builder that failed to
// destroy keeps its record, so `spekk sandbox destroy` can reach it.
func destroyBuilder(p Provider, name string, meta *SandboxMeta) {
	if !namesMachine(meta) {
		return
	}
	ref := machineRef(meta)
	if err := p.Destroy(meta); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not destroy builder %s: %s\nIt may still be billing. Remove it with: spekk sandbox destroy %s --force\n", ref, err, name)
		return
	}
	removeGeneratedKeyPair(meta.SSHKeyPath)
	if err := os.Remove(KnownHostsFile(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "Warning: could not remove the host key of builder %s: %s\n", name, err)
	}
	if err := RemoveSandbox(name); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not remove the record of builder %s: %s\n", name, err)
	}
	fmt.Fprintf(os.Stderr, "Removed builder %s, its record, its key pair, and its host key.\n", ref)
}

// runSetup is the seam the build runs the setup script through.
var runSetup = runSetupScript

// runSetupScript sends script to the builder on the stdin of ssh, runs it as
// root, and streams its output to the terminal.
func runSetupScript(meta *SandboxMeta, name, script string) error {
	shell := "bash -s"
	if sshUser(meta) != "root" {
		shell = "sudo bash -s"
	}
	cmd := exec.Command("ssh", append(sshBatchArgs(meta, name), shell)...)
	// As in runSecretScript: a step that reads stdin must not eat the rest
	// of the script.
	cmd.Stdin = strings.NewReader("{\n" + script + "\n} </dev/null\n")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// imageRoot is the root the hygiene script works on. Empty is the machine
// itself; a test sets a temporary directory, so the real script runs against
// planted files and never against the machine that runs the test.
var imageRoot = ""

// hygieneScript cleans the builder and then checks it, in one SSH session:
// the clean step removes the authorized keys, so a new session could not log
// in after it. Credentials are not removed: a check that finds one fails the
// build, because a setup script that left one can have left a copy that
// this script does not know about. Claude Code settings without a sign-in
// are removed, because the base cloud-init writes them. It prints "os=<PRETTY_NAME>" first and
// one "hygiene: <finding>" line for each finding, and exits 1 when there is
// one.
func hygieneScript(root string) string {
	return `R=` + shellQuote(root) + `
os=$( . "$R/etc/os-release" 2>/dev/null && echo "$PRETTY_NAME")
echo "os=$os"

# Clean: what makes each sandbox its own machine, and the logs.
rm -f "$R"/etc/ssh/ssh_host_*
if [ -f "$R/etc/machine-id" ]; then : > "$R/etc/machine-id"; fi
if [ ! -L "$R/var/lib/dbus/machine-id" ]; then rm -f "$R/var/lib/dbus/machine-id"; fi
rm -f "$R` + provisionedMarker + `"
# Each home: /root, each directory in /home, and the home of each user in
# /etc/passwd, so a service user with a home in /var/lib is not missed.
homes() {
  { echo /root; for d in "$R"/home/*; do echo "${d#"$R"}"; done; cut -d: -f6 "$R/etc/passwd" 2>/dev/null; } |
    while IFS= read -r h; do
      h=${h%/}
      case $h in "" | /dev | /proc | /sys | /run | /run/*) continue ;; esac
      if [ -d "$R$h" ]; then echo "$R$h"; fi
    done | awk '!seen[$0]++'
}
while IFS= read -r H; do
  rm -f "$H/.ssh/authorized_keys" "$H"/.*_history "$H/.viminfo"
  # The recipient key of the agent client makes the sandbox its own
  # machine, as a host key does: each sandbox must make its own.
  rm -rf "$H/.config/spekk-agent"
  # The Claude Code installer writes its settings. A sign-in in them is a
  # credential, so it stays, and the check below fails on it.
  if [ ! -e "$H/.claude/.credentials.json" ]; then rm -rf "$H/.claude" "$H/.claude.json"; fi
done < <(homes)
if [ -z "$R" ]; then
  cloud-init clean --logs
  journalctl --rotate >/dev/null 2>&1 || true
  journalctl --vacuum-time=1s >/dev/null 2>&1 || true
  find /var/log -type f ! -path '/var/log/journal/*' -exec truncate -s 0 {} + 2>/dev/null || true
fi

# Check.
found=0
bad() { echo "hygiene: $1"; found=1; }
for f in "$R/etc/spekk/agent.env" "$R/home/agent/.bashrc.d/spekk.sh" \
  "$R"/etc/ssh/ssh_host_* "$R` + provisionedMarker + `" "$R/var/lib/cloud/instance" \
  "$R"/var/log/cloud-init.log "$R"/var/log/cloud-init-output.log; do
  if [ -e "$f" ] || [ -L "$f" ]; then bad "${f#"$R"} is on the builder"; fi
done
while IFS= read -r H; do
  for f in "$H/.git-credentials" "$H/.config/git/credentials" "$H/.config/gh/hosts.yml" \
    "$H/.netrc" "$H/.aws/credentials" "$H/.config/gcloud" "$H/.config/doctl/config.yaml" \
    "$H/.claude/.credentials.json" "$H/.claude.json" "$H"/.*_history "$H/.viminfo"; do
    if [ -e "$f" ] || [ -L "$f" ]; then bad "${f#"$R"} is on the builder"; fi
  done
  # Any file in .ssh can be a private key, whatever its name. The slash
  # makes find follow a .ssh that is a symlink.
  if [ -d "$H/.ssh" ]; then
    while IFS= read -r f; do bad "${f#"$R"} is on the builder"; done < <(
      find "$H/.ssh/" -mindepth 1 ! -type d ! -name known_hosts ! -name config ! -name '*.pub' 2>/dev/null)
  fi
done < <(homes)
if [ -s "$R/etc/machine-id" ]; then bad "/etc/machine-id is not empty"; fi

# The content checks find a secret by its shape, wherever it is in these
# roots. Docker keeps the images it pulled in /var/lib/docker and
# /var/lib/containerd, so they are not searched.
roots=("$R/opt" "$R/srv")
while IFS= read -r H; do roots+=("$H"); done < <(homes)
for d in "$R"/var/lib/*; do
  case ${d#"$R"} in /var/lib/docker | /var/lib/containerd) continue ;; esac
  if [ -d "$d" ]; then roots+=("$d"); fi
done
# A key header ends its line, or is followed by an escaped newline and the
# key (a key in a JSON file). Code that only names the header does not
# match. Installed packages often ship test keys, so their directories are
# not searched. archive-v0 in the uv cache has the unpacked wheels.
key='-----BEGIN ([A-Z0-9]+ )?PRIVATE KEY-----([[:space:]]*$|\\n[A-Za-z0-9+/]{20})'
while IFS= read -r f; do bad "${f#"$R"} has a private key"; done < <(
  grep -rlIsE --exclude-dir=node_modules --exclude-dir=site-packages --exclude-dir=dist-packages --exclude-dir=archive-v0 \
    -e "$key" "${roots[@]}" | awk '!seen[$0]++')
# A token is a full word in its real shape, so a match inside base64 or a
# short test value is not a finding. AKIAIOSFODNN7EXAMPLE is the key in the
# AWS documentation.
token='(^|[^A-Za-z0-9_])(gh[pousr]_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{22,}|sk-ant-[a-z]+[0-9]{2}-[A-Za-z0-9_-]{80,}|AKIA[0-9A-Z]{16}|dop_v1_[a-f0-9]{64}|glpat-[A-Za-z0-9_-]{20}|xox[abprs]-[0-9]{6,}-[A-Za-z0-9-]{10,})([^A-Za-z0-9_]|$)'
while IFS= read -r f; do
  if grep -oIsE -e "$token" "$f" | grep -qvE 'EXAMPLE[^A-Za-z0-9_]?$'; then bad "${f#"$R"} has a token"; fi
done < <(grep -rlIsE -e "$token" "${roots[@]}" "$R/etc" | awk '!seen[$0]++')

# Each git config: a user or token in a URL, or an extra HTTP header (the
# form that a CI checkout writes), is a credential.
gitconfigs() {
  find "${roots[@]}" -path '*.git/config' -type f 2>/dev/null
  for H in "${roots[@]}"; do echo "$H/.gitconfig"; echo "$H/.config/git/config"; done
  echo "$R/etc/gitconfig"
}
while IFS= read -r f; do
  [ -f "$f" ] || continue
  if grep -Eq 'https?://[^/@[:space:]]+@' "$f"; then bad "${f#"$R"} has a credential in a remote URL"; fi
  if grep -iq 'extraheader' "$f"; then bad "${f#"$R"} has an extra HTTP header"; fi
done < <(gitconfigs | awk '!seen[$0]++')

for f in "$R"/root/.docker/config.json "$R"/home/*/.docker/config.json; do
  if [ -f "$f" ] && grep -q '"auth"' "$f"; then bad "${f#"$R"} has a registry login"; fi
done
for f in "$R"/root/.npmrc "$R"/home/*/.npmrc; do
  if [ -f "$f" ] && grep -q '_authToken' "$f"; then bad "${f#"$R"} has an npm token"; fi
done
secretvar='[A-Za-z_]*(TOKEN|SECRET|API_KEY|PASSWORD|ACCESS_KEY)[A-Za-z_]*=[^[:space:]]'
envfiles() {
  echo "$R/etc/environment"; echo "$R/etc/bash.bashrc"
  for f in "$R"/etc/profile.d/*; do echo "$f"; done
  while IFS= read -r H; do
    for f in .bashrc .profile .bash_profile .bash_login .bash_aliases .zshrc .zprofile .zshenv; do echo "$H/$f"; done
    for f in "$H"/.bashrc.d/*; do echo "$f"; done
  done < <(homes)
  find "${roots[@]}" -name .env -type f 2>/dev/null
}
while IFS= read -r f; do
  if [ -f "$f" ] && grep -Eq "^[[:space:]]*(export[[:space:]]+)?$secretvar" "$f"; then
    bad "${f#"$R"} sets a secret variable"
  fi
done < <(envfiles | awk '!seen[$0]++')
while IFS= read -r f; do bad "${f#"$R"} sets a secret variable"; done < <(
  grep -rlsE "^[[:space:]]*Environment=.*$secretvar" "$R/etc/systemd/system")
exit $found`
}

// shellQuote quotes s for a POSIX shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// checkHygiene runs the hygiene script on the builder as root, and returns
// the OS version it read. A finding is an error that lists each one.
func checkHygiene(meta *SandboxMeta, name string) (osVersion string, err error) {
	out, runErr := runSecretScript(meta.IP, meta.SSHKeyPath, name, sshUser(meta), hygieneScript(imageRoot))
	var findings []string
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(line, "os="); ok {
			osVersion = strings.TrimSpace(v)
		} else if f, ok := strings.CutPrefix(line, "hygiene: "); ok {
			findings = append(findings, "  - "+f)
		}
	}
	if len(findings) > 0 {
		return "", fmt.Errorf("the hygiene check failed, so no image was made:\n%s\nRemove these in the setup script, then build again", strings.Join(findings, "\n"))
	}
	if runErr != nil {
		return "", fmt.Errorf("the hygiene step failed: %w\n%s", runErr, strings.TrimSpace(out))
	}
	return osVersion, nil
}

// --- Create from an image ---

// applyImage resolves --image for a create. When the provider has the image
// in the region of the create, it sets opts.Image and the first-boot
// cloud-init. Otherwise it prints the fallback line, and the create uses the
// full cloud-init.
func applyImage(p Provider, opts *CreateOptions, now time.Time) error {
	if opts.ImageName == "" {
		return nil
	}
	if p == nil {
		return fmt.Errorf("--image needs a cloud provider: an existing machine (--ip) is not created from an image")
	}
	imager, ok := p.(Imager)
	region := ""
	var img *ImageMeta
	if ok {
		region = imager.ImageRegion(*opts)
		img = findImage(imager, p.Name(), opts.ImageName)
	}
	if img == nil || (region != "" && len(img.Regions) > 0 && !slices.Contains(img.Regions, region)) {
		where := p.Name()
		if region != "" {
			where += " in " + region
		}
		fmt.Fprintf(os.Stderr, "No image %q for %s; creating from the stock image with the full cloud-init.\n", opts.ImageName, where)
		return nil
	}
	if age := img.age(now); age > imageMaxAge {
		fmt.Fprintf(os.Stderr, "Warning: image %q is %d days old. Rebuild it for security updates: spekk sandbox image delete %s, then spekk sandbox image build --name %s with the same --setup\n",
			img.Name, int(age.Hours()/24), img.Name, img.Name)
	}
	fmt.Fprintf(os.Stderr, "Creating from image %q (%s).\n", img.Name, img.ImageID)
	opts.Image = img
	opts.CloudInit = imageCloudInit
	return nil
}

// findImage finds an image by name: first in images.json, then, for a
// caller with no local record such as a machine-mode worker, in the
// provider's own list. An error reads as "no image", so the create falls
// back.
func findImage(imager Imager, provider, name string) *ImageMeta {
	if images, err := LoadImages(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", err)
	} else if img, ok := images[name]; ok && img.Provider == provider {
		return img
	}
	listed, err := imager.ListImages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not list the %s images: %s\n", provider, err)
		return nil
	}
	for _, img := range listed {
		if img.Name == name {
			img.Provider = provider
			return &img
		}
	}
	return nil
}

// --- List and delete ---

// ListImages prints the image records.
func ListImages(out io.Writer, now time.Time) error {
	images, err := LoadImages()
	if err != nil {
		return err
	}
	if len(images) == 0 {
		fmt.Fprintln(out, "No images.")
		return nil
	}
	names := make([]string, 0, len(images))
	for name := range images {
		names = append(names, name)
	}
	sort.Strings(names)
	row := "%-20s %-13s %-16s %-12s %-20s %-5s %-9s %s\n"
	fmt.Fprintf(out, row, "NAME", "PROVIDER", "IMAGE ID", "REGIONS", "CREATED", "AGE", "SPEKK", "OS")
	for _, name := range names {
		img := images[name]
		age := "-"
		if d := img.age(now); d > 0 {
			age = fmt.Sprintf("%dd", int(d.Hours()/24))
		}
		regions := strings.Join(img.Regions, ",")
		fmt.Fprintf(out, row, name, img.Provider, img.ImageID, orDash(regions), orDash(img.CreatedAt), age, orDash(img.SpekkVersion), orDash(img.OSVersion))
	}
	return nil
}

// DeleteImage deletes the provider image and then the record. An image the
// provider no longer has only loses its record.
func DeleteImage(name string, providerFor func(string) (Provider, error)) error {
	images, err := LoadImages()
	if err != nil {
		return err
	}
	img, ok := images[name]
	if !ok {
		return fmt.Errorf("image %q not found; see spekk sandbox image list", name)
	}
	p, err := providerFor(img.Provider)
	if err != nil {
		return err
	}
	imager, ok := p.(Imager)
	if !ok {
		return fmt.Errorf("provider %s cannot delete images", img.Provider)
	}
	if err := imager.DeleteImage(*img); err != nil {
		if !errors.Is(err, ErrImageNotFound) {
			return fmt.Errorf("deleting image %s (%s): %w", name, img.ImageID, err)
		}
		fmt.Fprintf(os.Stderr, "Warning: %s has no image %s; removing the record.\n", img.Provider, img.ImageID)
	}
	delete(images, name)
	if err := saveImages(images); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Image %s (%s) deleted.\n", name, img.ImageID)
	return nil
}
