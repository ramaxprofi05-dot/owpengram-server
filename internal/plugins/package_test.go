package plugins

import (
	"archive/zip"
	"io/fs"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	assets := []byte("plugin icon asset")
	m.SHA256 = PackageContentSHA256(map[string][]byte{entryName: entrypoint, "assets/icon.bin": assets})
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
	for name, body := range map[string][]byte{pluginManifestName: manifestBytes, entryName: entrypoint, "assets/icon.bin": assets} {
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

	assetTampered := append([]byte(nil), data...)
	assetIndex := bytes.Index(assetTampered, []byte("plugin icon asset"))
	if assetIndex < 0 {
		t.Fatal("test asset not found in archive")
	}
	assetTampered[assetIndex] ^= 1
	if _, err := ValidatePackageArchive(assetTampered, trusted); err == nil {
		t.Fatal("modified non-entrypoint asset was accepted")
	}
}

func makeArchiveWithEntries(t *testing.T, entries []struct {
	name string
	mode fs.FileMode
	body []byte
}) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		if entry.mode != 0 {
			header.SetMode(entry.mode)
		}
		part, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create archive entry %q: %v", entry.name, err)
		}
		if _, err := part.Write(entry.body); err != nil {
			t.Fatalf("write archive entry %q: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return buffer.Bytes()
}

func TestValidatePackageArchiveRejectsUnsafeDuplicateAndSymlinkEntries(t *testing.T) {
	cases := []struct {
		name    string
		entries []struct {
			name string
			mode fs.FileMode
			body []byte
		}
	}{
		{
			name: "zip slip",
			entries: []struct {
				name string
				mode fs.FileMode
				body []byte
			}{{name: "../escape", body: []byte("x")}},
		},
		{
			name: "duplicate path",
			entries: []struct {
				name string
				mode fs.FileMode
				body []byte
			}{{name: "dist/main.js", body: []byte("one")}, {name: "dist/main.js", body: []byte("two")}},
		},
		{
			name: "symbolic link",
			entries: []struct {
				name string
				mode fs.FileMode
				body []byte
			}{{name: "dist/main.js", mode: fs.ModeSymlink | 0o777, body: []byte("../outside")}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := makeArchiveWithEntries(t, tc.entries)
			if _, err := ValidatePackageArchive(archive, nil); err == nil {
				t.Fatal("unsafe archive was accepted")
			}
		})
	}
}

func TestValidatePackageArchiveRejectsMalformedAndManifestlessArchives(t *testing.T) {
	if _, err := ValidatePackageArchive([]byte("not a zip archive"), nil); err == nil {
		t.Fatal("malformed ZIP was accepted")
	}

	var empty bytes.Buffer
	writer := zip.NewWriter(&empty)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePackageArchive(empty.Bytes(), nil); err == nil {
		t.Fatal("empty ZIP was accepted")
	}

	manifestless := makeArchiveWithEntries(t, []struct {
		name string
		mode fs.FileMode
		body []byte
	}{{name: "dist/main.js", body: []byte("export default {};")}})
	if _, err := ValidatePackageArchive(manifestless, nil); err == nil {
		t.Fatal("archive without root manifest.json was accepted")
	}
}

func TestValidatePackageArchiveEnforcesEntryCountLimit(t *testing.T) {
	entries := make([]struct {
		name string
		mode fs.FileMode
		body []byte
	}, MaxPackageFiles+1)
	for i := range entries {
		entries[i] = struct {
			name string
			mode fs.FileMode
			body []byte
		}{name: fmt.Sprintf("assets/%03d.bin", i), body: []byte{byte(i)}}
	}
	archive := makeArchiveWithEntries(t, entries)
	if _, err := ValidatePackageArchive(archive, nil); err == nil {
		t.Fatalf("archive with %d entries exceeded limit %d but was accepted", len(entries), MaxPackageFiles)
	}
}

}
