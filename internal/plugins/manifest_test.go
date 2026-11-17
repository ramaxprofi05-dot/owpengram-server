package plugins

import (
	"encoding/base64"
	"strings"
	"testing"
)

func validManifest() Manifest {
	return Manifest{
		ID:          "dev.femboygram.weather",
		Name:        "Weather",
		Version:     "1.2.3",
		Author:      "FemboyGram Community",
		Description: "Shows a small weather card.",
		Target:      TargetAndroid,
		Entrypoint:  "dist/main.js",
		Permissions: []string{"ui:chat-list", "network:https"},
	}
}

func TestValidateManifestAcceptsLeastPrivilegeManifest(t *testing.T) {
	if err := ValidateManifest(validManifest()); err != nil {
		t.Fatalf("ValidateManifest: %v", err)
	}
}

func TestValidateManifestRejectsUnsafeOrUnknownPermissions(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Manifest)
	}{
		{"arbitrary filesystem", func(m *Manifest) { m.Permissions = []string{"filesystem:full"} }},
		{"session tokens", func(m *Manifest) { m.Permissions = []string{"session:tokens"} }},
		{"duplicate permission", func(m *Manifest) { m.Permissions = []string{"ui:chat-list", "ui:chat-list"} }},
		{"path traversal", func(m *Manifest) { m.Entrypoint = "../main.js" }},
		{"absolute path", func(m *Manifest) { m.Entrypoint = "/tmp/main.js" }},
		{"wrong runtime extension", func(m *Manifest) { m.Target = TargetServer }},
		{"bad semver", func(m *Manifest) { m.Version = "latest" }},
		{"bad reverse domain", func(m *Manifest) { m.ID = "weather" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validManifest()
			tc.edit(&m)
			if err := ValidateManifest(m); err == nil {
				t.Fatal("unsafe manifest was accepted")
			}
		})
	}
}

func TestParseManifestRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	base := `{"id":"dev.example.weather","name":"Weather","version":"1.0.0","author":"Example","description":"Weather card","target":"android","entrypoint":"main.js","permissions":[]}`
	if _, err := ParseManifest([]byte(strings.Replace(base, "\"name\":", "\"unreviewed_permission\":", 1))); err == nil {
		t.Fatal("unknown field was accepted")
	}
	if _, err := ParseManifest([]byte(base + base)); err == nil {
		t.Fatal("trailing JSON object was accepted")
	}
}

func TestValidateInstallableManifestRequiresIntegrityMetadata(t *testing.T) {
	m := validManifest()
	if err := ValidateInstallableManifest(m); err == nil {
		t.Fatal("unsigned draft was accepted as installable")
	}
	m.SHA256 = strings.Repeat("a", 64)
	m.SignerKeyID = "femboygram-community-key-01"
	m.Signature = base64.StdEncoding.EncodeToString(make([]byte, 64))
	if err := ValidateInstallableManifest(m); err != nil {
		t.Fatalf("signed manifest: %v", err)
	}
}

func TestSafeArchivePathRejectsZipSlip(t *testing.T) {
	for _, name := range []string{"../escape", "a/../../escape", "/absolute", "a\\b", ".", ""} {
		if SafeArchivePath(name) {
			t.Fatalf("unsafe archive path %q accepted", name)
		}
	}
	for _, name := range []string{"manifest.json", "dist/main.js", "assets/icon.png"} {
		if !SafeArchivePath(name) {
			t.Fatalf("safe archive path %q rejected", name)
		}
	}
}
