package plugins

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
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

func TestValidateManifestAcceptsProfileBannerCapability(t *testing.T) {
	m := validManifest()
	m.Permissions = []string{"ui:profile", "ui:profile-banner"}
	if err := ValidateManifest(m); err != nil {
		t.Fatalf("profile banner capability should be allowed for Android: %v", err)
	}
	m.Target = TargetServer
	if err := ValidateManifest(m); err == nil {
		t.Fatal("Android profile banner capability must not be allowed for server plugins")
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
		{"numeric prerelease leading zero", func(m *Manifest) { m.Version = "1.0.0-01" }},
		{"empty prerelease identifier", func(m *Manifest) { m.Version = "1.0.0-alpha..1" }},
		{"empty build identifier", func(m *Manifest) { m.Version = "1.0.0+build..x" }},
		{"numeric prerelease identifier leading zero", func(m *Manifest) { m.Version = "2.4.0-rc.01" }},
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
	if _, err := ParseManifest([]byte(base + " trailing")); err == nil {
		t.Fatal("trailing non-JSON data was accepted")
	}
}

func TestParseManifestRejectsDuplicateJSONKeys(t *testing.T) {
	base := `{"id":"dev.example.weather","name":"Weather","version":"1.0.0","author":"Example","description":"Weather card","target":"android","entrypoint":"main.js","permissions":[]}`
	cases := []struct {
		name string
		data string
	}{
		{"duplicate top-level field", strings.Replace(base, `"name":"Weather"`, `"name":"Weather","name":"Overridden"`, 1)},
		{"escaped duplicate top-level field", strings.Replace(base, `"name":"Weather"`, `"name":"Weather","\u006eame":"Overridden"`, 1)},
		{"duplicate nested field", `{"id":"dev.example.weather","name":"Weather","version":"1.0.0","author":"Example","description":"Weather card","target":"android","entrypoint":"main.js","permissions":[],"metadata":{"x":1,"x":2}}`},
		{"multiple top-level values", base + ` {}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseManifest([]byte(tc.data)); err == nil {
				t.Fatal("ambiguous JSON was accepted")
			}
		})
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

func TestVerifyPluginPackageUsesArchiveContentDigestAndTrustedSignature(t *testing.T) {
	archive, publicKey, manifest := signedTestArchive(t, "dist/main.js", []byte("export default {};"))
	trusted := map[string]ed25519.PublicKey{manifest.SignerKeyID: publicKey}

	got, err := VerifyPluginPackage(archive, trusted)
	if err != nil {
		t.Fatalf("valid package rejected: %v", err)
	}
	if got.ID != manifest.ID || got.SHA256 != manifest.SHA256 {
		t.Fatalf("unexpected verified manifest: %#v", got)
	}
	if _, err := VerifyPluginPackage(archive, nil); err == nil {
		t.Fatal("package signed by an untrusted publisher was accepted")
	}

	tampered := append([]byte(nil), archive...)
	index := bytes.Index(tampered, []byte("export default {};"))
	if index < 0 {
		t.Fatal("test entrypoint not found in archive")
	}
	tampered[index] ^= 1
	if _, err := VerifyPluginPackage(tampered, trusted); err == nil {
		t.Fatal("package with modified entrypoint was accepted")
	}
}

func TestSafeArchivePathRejectsZipSlip(t *testing.T) {
	for _, name := range []string{"../escape", "a/../../escape", "/absolute", "a\\b", "C:/escape", "nul\x00file", ".", ""} {
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


func TestVerifyManifestSignatureRequiresTrustedPublisherAndExactPayload(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}

	m := validManifest()
	m.SHA256 = strings.Repeat("b", 64)
	m.SignerKeyID = "femboygram-test-key"
	payload, err := SigningPayload(m)
	if err != nil {
		t.Fatalf("SigningPayload: %v", err)
	}
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))

	trusted := map[string]ed25519.PublicKey{m.SignerKeyID: publicKey}
	if err := VerifyManifestSignature(m, trusted); err != nil {
		t.Fatalf("valid trusted signature rejected: %v", err)
	}
	if err := VerifyManifestSignature(m, nil); err == nil {
		t.Fatal("signature from an unknown publisher was accepted")
	}

	tampered := m
	tampered.Description = "A different package description"
	if err := VerifyManifestSignature(tampered, trusted); err == nil {
		t.Fatal("manifest changed after signing was accepted")
	}
}
