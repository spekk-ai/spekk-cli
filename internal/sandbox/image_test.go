package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeImager is a provider that can make images, and records what it did.
type fakeImager struct {
	recordingProvider
	made      []string // the names CreateImage was asked for
	destroyed int
}

func (f *fakeImager) Destroy(meta *SandboxMeta) error {
	f.destroyed++
	return nil
}

func (f *fakeImager) CreateImage(meta *SandboxMeta, name string, regions []string) (ImageMeta, error) {
	f.made = append(f.made, name)
	return ImageMeta{ImageID: "9001", Regions: regions}, nil
}

func (f *fakeImager) DeleteImage(ImageMeta) error           { return nil }
func (f *fakeImager) ListImages() ([]ImageMeta, error)      { return nil, nil }
func (f *fakeImager) ImageRegion(opts CreateOptions) string { return orDefault(opts.Region, "nyc1") }

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// useTempImages points images.json at a temp dir.
func useTempImages(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "images.json")
	orig := imagesFile
	imagesFile = func() string { return path }
	t.Cleanup(func() { imagesFile = orig })
}

// builderOnDisk makes a temporary root that stands for the builder, and an
// SSH runner that runs each script from the build on it with the real bash.
// The runner refuses a script that is not for that root, so a mistake in a
// test can never clean the machine that runs it.
func builderOnDisk(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"etc/ssh", "etc/spekk", "home/agent/.ssh", "opt/spekk", "root/.ssh"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, content := range map[string]string{
		"etc/os-release":                   `PRETTY_NAME="Ubuntu 24.04.3 LTS"` + "\n",
		"etc/machine-id":                   "0123456789abcdef0123456789abcdef\n",
		"etc/ssh/ssh_host_ed25519_key":     "builder host key\n",
		"home/agent/.ssh/authorized_keys":  "ssh-ed25519 AAAA builder\n",
		"home/agent/.ssh/known_hosts":      "github.com ssh-ed25519 AAAA\n",
		"home/agent/.ssh/id_ed25519.pub":   "ssh-ed25519 AAAA agent\n",
		"home/agent/.bash_history":         "ls\n",
		"opt/spekk/.provisioned":           "",
		"home/agent/.bashrc":               "export PATH=$PATH:/opt/bin\nGITHUB_TOKEN=\n",
		"root/.ssh/authorized_keys":        "ssh-ed25519 AAAA builder\n",
		"home/agent/.docker/config.json":   `{"credsStore": "desktop"}`,
		"home/agent/.claude/settings.json": `{"autoUpdatesChannel": "stable"}`,
		// Not secrets: code that names a key header, a test key of an
		// installed package, and the key of the AWS documentation.
		"opt/app/keys.py":                       `START = b"-----BEGIN OPENSSH PRIVATE KEY-----"` + "\n",
		"opt/app/node_modules/x/test/key.pem":   "-----BEGIN RSA PRIVATE KEY-----\nMIIB\n",
		"home/agent/.local/lib/aws/example.txt": "aws_access_key_id = AKIAIOSFODNN7EXAMPLE\n",
	} {
		full := filepath.Join(root, path)
		os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	origRoot, origSSH, origWait, origSetup := imageRoot, sshExec, waitReady, runSetup
	t.Cleanup(func() { imageRoot, sshExec, waitReady, runSetup = origRoot, origSSH, origWait, origSetup })
	imageRoot = root
	sshExec = func(args []string, stdin string) ([]byte, error) {
		if !strings.Contains(stdin, "R='"+root+"'") {
			t.Fatalf("a script for another root reached the fake builder:\n%s", stdin)
		}
		cmd := exec.Command("bash", "-s")
		cmd.Stdin = strings.NewReader(stdin)
		return cmd.CombinedOutput()
	}
	waitReady = func(ip, keyPath, name, user string, timeout time.Duration) error { return nil }
	runSetup = func(meta *SandboxMeta, name, script string) error { return nil }
	return root
}

// The build cleans the builder and checks it in one session. A clean builder
// becomes an image with a full record. A credential that the setup left
// fails the build before any image exists. Each way, the builder goes.
func TestBuildImage(t *testing.T) {
	for _, tc := range []struct {
		name    string
		plant   string // a file the setup left on the builder
		content string // of plant; a token line when empty
		prep    func(root string)
		wantErr string
	}{
		{name: "a clean builder becomes an image"},
		{name: "a credential file fails the build", plant: "etc/spekk/agent.env", wantErr: "/etc/spekk/agent.env is on the builder"},
		{name: "a token in a shell file fails the build", plant: "etc/profile.d/ci.sh", wantErr: "/etc/profile.d/ci.sh sets a secret variable"},
		{name: "a Claude Code sign-in of root fails the build", plant: "root/.claude/.credentials.json", wantErr: "/root/.claude/.credentials.json is on the builder"},
		{name: "an SSH private key fails the build", plant: "home/agent/.ssh/id_ed25519", wantErr: "/home/agent/.ssh/id_ed25519 is on the builder"},
		{name: "a token in a remote URL fails the build", plant: "opt/app/.git/config", content: "[remote \"origin\"]\n\turl = https://x-access-token:ghp_x@github.com/acme/app.git\n", wantErr: "/opt/app/.git/config has a credential in a remote URL"},
		{name: "any shell history is removed", plant: "home/agent/.python_history"},
		{name: "a home from /etc/passwd outside /home is checked", plant: "var/ci/.git-credentials", content: "https://u:p@example.com\n",
			prep: func(root string) {
				os.WriteFile(filepath.Join(root, "etc/passwd"), []byte("root:x:0:0:root:/root:/bin/bash\nci:x:999:999::/var/ci:/bin/sh\n"), 0o644)
			}, wantErr: "/var/ci/.git-credentials is on the builder"},
		{name: "a .ssh that is a symlink is searched", plant: "keys/id_rsa", content: "key\n",
			prep: func(root string) {
				os.MkdirAll(filepath.Join(root, "home/y"), 0o755)
				os.Symlink("../../keys", filepath.Join(root, "home/y/.ssh"))
			}, wantErr: "/home/y/.ssh/id_rsa is on the builder"},
		{name: "a token in a known format fails the build", plant: "opt/app/notes.txt", content: "deploy with ghp_" + strings.Repeat("a1B2", 9) + "\n", wantErr: "/opt/app/notes.txt has a token"},
		{name: "a private key in a JSON file fails the build", plant: "srv/app/sa.json", content: `{"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASC\n-----END PRIVATE KEY-----\n"}`, wantErr: "/srv/app/sa.json has a private key"},
		{name: "an extra HTTP header in a git config fails the build", plant: "srv/app/.git/config", content: "[http \"https://github.com/\"]\n\textraHeader = AUTHORIZATION: basic eDp5\n", wantErr: "/srv/app/.git/config has an extra HTTP header"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			useTempStore(t)
			useTempImages(t)
			root := builderOnDisk(t)
			if tc.plant != "" {
				full := filepath.Join(root, tc.plant)
				os.MkdirAll(filepath.Dir(full), 0o755)
				content := tc.content
				if content == "" {
					content = "export SPEKK_AGENT_TOKEN=s3cret\n"
				}
				os.WriteFile(full, []byte(content), 0o600)
			}
			if tc.prep != nil {
				tc.prep(root)
			}
			setup := filepath.Join(t.TempDir(), "setup.sh")
			os.WriteFile(setup, []byte("echo setup\n"), 0o644)

			p := &fakeImager{recordingProvider: recordingProvider{ip: "203.0.113.5", dropletID: 77}}
			captureStderr(t)
			err := BuildImage(p, ImageBuildOptions{Name: "web-app", Setup: setup, Regions: []string{"nyc1", "nyc3"}})

			if p.destroyed != 1 {
				t.Errorf("builder destroyed %d times, want 1", p.destroyed)
			}
			if got, _ := GetSandbox("image-web-app"); got != nil {
				t.Errorf("the builder record stays: %+v", got)
			}
			images, _ := LoadImages()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				if len(p.made) != 0 || len(images) != 0 {
					t.Errorf("an image was made after a failed check: %v %v", p.made, images)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			img := images["web-app"]
			if img == nil || img.Provider != "digitalocean" || img.ImageID != "9001" || strings.Join(img.Regions, ",") != "nyc1,nyc3" ||
				img.OSVersion != "Ubuntu 24.04.3 LTS" || len(img.SetupSHA256) != 64 || img.CreatedAt == "" || img.SpekkVersion == "" {
				t.Fatalf("image record = %+v", img)
			}
			for _, gone := range []string{"etc/ssh/ssh_host_ed25519_key", "home/agent/.ssh/authorized_keys", "home/agent/.bash_history", "opt/spekk/.provisioned", "home/agent/.claude", tc.plant} {
				if gone == "" {
					continue
				}
				if _, err := os.Stat(filepath.Join(root, gone)); err == nil {
					t.Errorf("%s is still on the builder", gone)
				}
			}
			if id, _ := os.ReadFile(filepath.Join(root, "etc/machine-id")); len(id) != 0 {
				t.Errorf("machine-id = %q, want empty", id)
			}
		})
	}
}

// A rebuild under the same name must not meet the host key of the last
// builder of that name: each probe would fail on it until the timeout.
func TestBuildImageForgetsTheOldHostKey(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	useTempImages(t)
	builderOnDisk(t)
	kh := KnownHostsFile("image-web-app")
	os.MkdirAll(filepath.Dir(kh), 0o700)
	os.WriteFile(kh, []byte("203.0.113.5 ssh-ed25519 OLD\n"), 0o600)
	waitReady = func(ip, keyPath, name, user string, timeout time.Duration) error {
		if b, _ := os.ReadFile(kh); strings.Contains(string(b), "OLD") {
			t.Error("the new builder met the old host key")
		}
		// The first SSH session records the host key of the new builder.
		return os.WriteFile(kh, []byte("203.0.113.5 ssh-ed25519 NEW\n"), 0o600)
	}

	p := &fakeImager{recordingProvider: recordingProvider{ip: "203.0.113.5", dropletID: 77}}
	captureStderr(t)
	if err := BuildImage(p, ImageBuildOptions{Name: "web-app"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(kh); err == nil {
		t.Error("the known_hosts file of the builder stays after the build")
	}
}

// create --image uses the image when the provider has it in the region of
// the create, and otherwise says so in one line and uses the full
// cloud-init.
func TestApplyImage(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-24 * time.Hour).Format(time.RFC3339)
	for _, tc := range []struct {
		name      string
		record    *ImageMeta // in images.json
		region    string
		wantImage bool
		wantLine  string
	}{
		{name: "record in the region", record: &ImageMeta{Provider: "digitalocean", ImageID: "1", Regions: []string{"nyc1", "nyc3"}, CreatedAt: fresh}, region: "nyc3", wantImage: true, wantLine: `Creating from image "base"`},
		{name: "record in other regions", record: &ImageMeta{Provider: "digitalocean", ImageID: "1", Regions: []string{"nyc1"}, CreatedAt: fresh}, region: "sfo3", wantLine: `No image "base" for digitalocean in sfo3; creating from the stock image with the full cloud-init.`},
		{name: "record of another provider", record: &ImageMeta{Provider: "gcp", ImageID: "base", CreatedAt: fresh}, wantLine: `No image "base" for digitalocean in nyc1`},
		{name: "an old image warns", record: &ImageMeta{Provider: "digitalocean", ImageID: "1", Regions: []string{"nyc1"}, CreatedAt: now.Add(-20 * 24 * time.Hour).Format(time.RFC3339)}, wantImage: true, wantLine: `image "base" is 20 days old`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			useTempImages(t)
			if tc.record != nil {
				tc.record.Name = "base"
				if err := SaveImage(*tc.record); err != nil {
					t.Fatal(err)
				}
			}
			p := &fakeImager{}
			opts := CreateOptions{ImageName: "base", Region: tc.region, CloudInit: []byte("#full")}
			stderr := captureStderr(t)
			if err := applyImage(p, &opts, now); err != nil {
				t.Fatal(err)
			}
			out := stderr()
			if got := opts.Image != nil; got != tc.wantImage {
				t.Errorf("image used = %v, want %v (stderr %q)", got, tc.wantImage, out)
			}
			if tc.wantImage != (string(opts.CloudInit) == string(imageCloudInit)) {
				t.Errorf("cloud-init does not match the image choice: %q", opts.CloudInit)
			}
			if !strings.Contains(out, tc.wantLine) {
				t.Errorf("stderr = %q, want %q", out, tc.wantLine)
			}
		})
	}
}
