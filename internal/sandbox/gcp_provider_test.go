package sandbox

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeGcloud replaces runGcloud with respond for one test and returns the
// argument lists of every call, joined with spaces.
func fakeGcloud(t *testing.T, respond func(cmd string) (string, error)) *[]string {
	t.Helper()
	var calls []string
	orig := runGcloud
	runGcloud = func(args ...string) (string, error) {
		cmd := strings.Join(args, " ")
		calls = append(calls, cmd)
		return respond(cmd)
	}
	t.Cleanup(func() { runGcloud = orig })
	return &calls
}

// A create through the generic layer records everything destroy needs to
// find the VM after the operator's gcloud configuration changes, and the
// provisioning wait logs in as the user the image admits.
func TestGCPCreateRecordsTheVMAndWaitsAsTheLoginUser(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	calls := fakeGcloud(t, func(cmd string) (string, error) {
		switch {
		case cmd == "config get-value project":
			return "proj-a", nil
		case cmd == "config get-value compute/zone":
			return "", nil
		case strings.HasPrefix(cmd, "compute instances create"):
			return "203.0.113.7", nil
		}
		return "", fmt.Errorf("unexpected gcloud %s", cmd)
	})
	var waitUser string
	orig := waitReady
	waitReady = func(ip, keyPath, name, user string, timeout time.Duration) error {
		waitUser = user
		return errors.New("stop here")
	}
	t.Cleanup(func() { waitReady = orig })

	if err := Create(&GCPProvider{}, CreateOptions{Name: "box"}); err == nil {
		t.Fatal("expected the stubbed wait to stop the create")
	}

	meta, err := GetSandbox("box")
	if err != nil || meta == nil {
		t.Fatalf("the VM must be recorded: %v", err)
	}
	want := SandboxMeta{Provider: ProviderGCP, GCPInstance: "spekk-box", Project: "proj-a", Region: gcpDefaultZone, Size: gcpDefaultMachineType, IP: "203.0.113.7", SSHUser: gcpLoginUser}
	got := SandboxMeta{Provider: meta.Provider, GCPInstance: meta.GCPInstance, Project: meta.Project, Region: meta.Region, Size: meta.Size, IP: meta.IP, SSHUser: meta.SSHUser}
	if got != want {
		t.Errorf("recorded %+v, want %+v", got, want)
	}
	if !ownsKeyPair(meta.SSHKeyPath) {
		t.Errorf("key %q is not a generated key, so destroy would leave it", meta.SSHKeyPath)
	}
	if waitUser != gcpLoginUser {
		t.Errorf("wait logged in as %q, want %q", waitUser, gcpLoginUser)
	}

	var create string
	for _, call := range *calls {
		if strings.HasPrefix(call, "compute instances create ") {
			create = call
		}
	}
	for _, part := range []string{
		"create spekk-box --project proj-a --zone " + gcpDefaultZone,
		"--image-family " + gcpImageFamily,
		"--no-service-account --no-scopes",
		"block-project-ssh-keys=TRUE,ssh-keys=spekk:ssh-ed25519 ",
		"--metadata-from-file user-data=",
	} {
		if !strings.Contains(create, part) {
			t.Errorf("create command lacks %q: %s", part, create)
		}
	}
}

// A failed create records the VM name only when the VM exists, so destroy
// can reach a VM that gcloud made before it failed, and is not sent after
// one that was never made.
func TestGCPCreateFailure(t *testing.T) {
	for _, tc := range []struct {
		name       string
		project    string
		create     error
		describe   error
		wantVM     string
		wantCreate bool
	}{
		{name: "no project", project: "", wantCreate: false},
		{name: "a project name is not an ID", project: "My Project", wantCreate: false},
		{name: "nothing made", project: "proj-a", describe: errors.New("The resource 'spekk-box' was not found"), wantCreate: true},
		{name: "VM made before the failure", project: "proj-a", wantVM: "spekk-box", wantCreate: true},
		{name: "invalid zone makes no VM", project: "proj-a", describe: errors.New("Invalid value for field 'zone': 'us-west1'. Unknown zone."), wantCreate: true},
		{name: "existence unknown keeps the record", project: "proj-a", describe: errors.New("permission denied"), wantVM: "spekk-box", wantCreate: true},
		{name: "a VM spekk did not make is not claimed", project: "proj-a", create: errors.New("The resource 'spekk-box' already exists"), wantCreate: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			calls := fakeGcloud(t, func(cmd string) (string, error) {
				switch {
				case cmd == "config get-value project":
					return tc.project, nil
				case strings.HasPrefix(cmd, "compute instances create"):
					if tc.create != nil {
						return "", tc.create
					}
					return "", errors.New("operation timed out")
				case strings.HasPrefix(cmd, "compute instances describe"):
					return "spekk-box", tc.describe
				}
				return "", fmt.Errorf("unexpected gcloud %s", cmd)
			})

			meta := &SandboxMeta{}
			if err := (&GCPProvider{}).Create("box", CreateOptions{Region: "z"}, meta); err == nil {
				t.Fatal("expected an error")
			}
			if meta.GCPInstance != tc.wantVM {
				t.Errorf("GCPInstance = %q, want %q", meta.GCPInstance, tc.wantVM)
			}
			if namesMachine(meta) != (tc.wantVM != "") {
				t.Errorf("namesMachine = %v for %+v", namesMachine(meta), meta)
			}
			created := strings.Contains(strings.Join(*calls, "\n"), "compute instances create")
			if created != tc.wantCreate {
				t.Errorf("create called = %v, want %v", created, tc.wantCreate)
			}
		})
	}
}

func TestGCPDestroy(t *testing.T) {
	vm := &SandboxMeta{GCPInstance: "spekk-box", Project: "proj-a", Region: "us-west1-b"}
	for _, tc := range []struct {
		name    string
		meta    *SandboxMeta
		gcloud  error
		wantErr error
		failed  bool
	}{
		{name: "deletes the VM", meta: vm},
		{name: "already gone is done", meta: vm, gcloud: errors.New("The resource 'spekk-box' was not found")},
		{name: "other failure stops", meta: vm, gcloud: errors.New("permission denied"), failed: true},
		{name: "no VM recorded refuses", meta: &SandboxMeta{}, wantErr: ErrNoMachineRecorded, failed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := fakeGcloud(t, func(string) (string, error) { return "", tc.gcloud })
			err := (&GCPProvider{}).Destroy(tc.meta)
			if (err != nil) != tc.failed || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
				t.Fatalf("Destroy() = %v", err)
			}
			if tc.meta.GCPInstance == "" {
				return
			}
			want := "compute instances delete spekk-box --project proj-a --zone us-west1-b --delete-disks boot --quiet"
			if len(*calls) != 1 || (*calls)[0] != want {
				t.Errorf("gcloud calls %q, want [%q]", *calls, want)
			}
		})
	}
}

// Without gcloud the provider cannot be built, and the caller must get an
// untyped nil, or the Status fallback calls a nil pointer.
func TestProviderByNameGCPWithoutGcloud(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, err := ProviderByName(ProviderGCP)
	if err == nil || !strings.Contains(err.Error(), "Google Cloud CLI") {
		t.Errorf("err = %v, want an install hint", err)
	}
	if p != nil {
		t.Errorf("p = %#v, want an untyped nil", p)
	}
}

// GCP images go through gcloud: the build stops the VM and makes a labeled
// image of its boot disk, a create from the image names it and its project
// instead of the Ubuntu family, the list reads the labeled images, and a
// delete of an image that is gone reads as not found.
func TestGCPImages(t *testing.T) {
	isolateConfig(t)
	calls := fakeGcloud(t, func(cmd string) (string, error) {
		switch {
		case cmd == "config get-value project":
			return "proj-a", nil
		case strings.HasPrefix(cmd, "compute instances create"):
			return "203.0.113.7", nil
		case strings.HasPrefix(cmd, "compute images list"):
			return `[{"name": "web-app", "creationTimestamp": "2026-10-10T08:00:00.000-07:00"}]`, nil
		case strings.HasPrefix(cmd, "compute images delete gone"):
			return "", errors.New("The resource 'projects/proj-a/global/images/gone' was not found")
		}
		return "", nil
	})
	p := &GCPProvider{}
	meta := &SandboxMeta{GCPInstance: "spekk-image-web-app", Project: "proj-a", Region: "us-east1-b"}

	img, err := p.CreateImage(meta, "web-app", nil)
	if err != nil || img.ImageID != "web-app" || img.Project != "proj-a" || len(img.Regions) != 0 {
		t.Fatalf("CreateImage = %+v, %v", img, err)
	}
	if err := p.Create("box", CreateOptions{Image: &ImageMeta{Name: "web-app", ImageID: "web-app", Project: "proj-a"}}, &SandboxMeta{}); err != nil {
		t.Fatal(err)
	}
	listed, err := p.ListImages()
	if err != nil || len(listed) != 1 || listed[0].Name != "web-app" || listed[0].CreatedAt != "2026-10-10T15:00:00Z" {
		t.Fatalf("ListImages = %+v, %v", listed, err)
	}
	if err := p.DeleteImage(ImageMeta{ImageID: "gone", Project: "proj-a"}); !errors.Is(err, ErrImageNotFound) {
		t.Errorf("DeleteImage of a missing image = %v, want ErrImageNotFound", err)
	}

	got := strings.Join(*calls, "\n")
	if strings.Contains(got, "--image-family") {
		t.Errorf("a create from an image still names the Ubuntu family:\n%s", got)
	}
}
