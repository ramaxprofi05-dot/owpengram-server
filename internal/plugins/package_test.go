package plugins

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func signedTestArchive(t *testing.T, entryName string, entrypoint []byte) ([]byte, ed25519.PublicKey, Manifest) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	m := validManifest()
	m.Entrypoint = entryName
	sum := sha256.Sum256(entrypoint)
	m.SHA256 = hex.EncodeToString(sum[:])
	m.SignerKeyID = "femboygram-test-key"
	payload, err := SigningPayload(m)
	if err != nil {
		t.Fatal(err)
	}
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	manifestBytes, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range map[string][]byte{pluginManifestName: manifestBytes, entryName: entrypoint} {
		part, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes(), publicKey, m
}

func TestValidatePackageArchiveAcceptsTrustedSignedPackage(t *testing.T) {
	data, publicKey, expected := signedTestArchive(t, "dist/main.js", []byte("export default {};"))
	got, err := ValidatePackageArchive(data, map[string]ed25519.PublicKey{expected.SignerKeyID: publicKey})
	if err != nil {
		t.Fatalf("ValidatePackageArchive: %v", err)
	}
	if got.ID != expected.ID || got.Version != expected.Version {
		t.Fatalf("unexpected manifest returned: %#v", got)
	}
}

func TestValidatePackageArchiveRejectsUntrustedOrTamperedPackage(t *testing.T) {
	data, publicKey, manifest := signedTestArchive(t, "dist/main.js", []byte("export default {};"))
	if _, err := ValidatePackageArchive(data, nil); err == nil {
		t.Fatal("untrusted publisher was accepted")
	}
	trusted := map[string]ed25519.PublicKey{manifest.SignerKeyID: publicKey}

	tampered := append([]byte(nil), data...)
	// Change a byte in the stored entrypoint while leaving its signed digest intact.
	index := bytes.Index(tampered, []byte("export default {};"))
	if index < 0 {
		t.Fatal("test entrypoint not found in archive")
	}
	tampered[index] ^= 1
	if _, err := ValidatePackageArchive(tampered, trusted); err == nil {
		t.Fatal("modified entrypoint was accepted")
	}
}

