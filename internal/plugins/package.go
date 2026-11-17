package plugins

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
)

const (
	MaxPackageBytes       = 32 << 20
	MaxPackageFiles       = 256
	MaxPackageEntryBytes  = 16 << 20
	MaxPackageTotalBytes  = 64 << 20
	pluginManifestName    = "manifest.json"
)

// ValidatePackageArchive validates an uploaded ZIP without extracting it. The
// manifest's SHA256 is the digest of its declared entrypoint. The manifest
// signature is checked against the caller's explicit publisher trust store.
// No package code is executed by this function.
func ValidatePackageArchive(data []byte, trustedKeys map[string]ed25519.PublicKey) (Manifest, error) {
	if len(data) == 0 || len(data) > MaxPackageBytes {
		return Manifest{}, fmt.Errorf("plugin archive must be between 1 and %d bytes", MaxPackageBytes)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Manifest{}, fmt.Errorf("open plugin archive: %w", err)
	}
	if len(reader.File) == 0 || len(reader.File) > MaxPackageFiles {
		return Manifest{}, fmt.Errorf("plugin archive must contain 1-%d files", MaxPackageFiles)
	}

	files := make(map[string]*zip.File, len(reader.File))
	var expanded int64
	for _, file := range reader.File {
		name := file.Name
		if file.FileInfo().IsDir() {
			name = strings.TrimSuffix(name, "/")
			if !SafeArchivePath(name) {
				return Manifest{}, fmt.Errorf("unsafe plugin archive directory %q", file.Name)
			}
			continue
		}
		if !SafeArchivePath(name) {
			return Manifest{}, fmt.Errorf("unsafe plugin archive path %q", name)
		}
		if file.Mode()&fs.ModeSymlink != 0 {
			return Manifest{}, fmt.Errorf("plugin archive contains a symbolic link %q", name)
		}
		if _, duplicate := files[name]; duplicate {
			return Manifest{}, fmt.Errorf("plugin archive contains duplicate path %q", name)
		}
		if file.UncompressedSize64 > MaxPackageTotalBytes {
			return Manifest{}, fmt.Errorf("plugin archive entry %q is too large", name)
		}
		expanded += int64(file.UncompressedSize64)
		if expanded > MaxPackageTotalBytes {
			return Manifest{}, errors.New("plugin archive expands beyond the size limit")
		}
		files[name] = file
	}

	manifestFile, ok := files[pluginManifestName]
	if !ok || manifestFile.UncompressedSize64 > MaxManifestBytes {
		return Manifest{}, errors.New("plugin archive must contain a root manifest.json")
	}
	manifestBytes, err := readZipEntry(manifestFile, MaxManifestBytes)
	if err != nil {
		return Manifest{}, fmt.Errorf("read plugin manifest: %w", err)
	}
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return Manifest{}, err
	}
	if err := ValidateInstallableManifest(manifest); err != nil {
		return Manifest{}, err
	}

	if _, ok := files[manifest.Entrypoint]; !ok {
		return Manifest{}, fmt.Errorf("plugin entrypoint %q is missing from archive", manifest.Entrypoint)
	}
	packageFiles := make(map[string][]byte, len(files)-1)
	for name, file := range files {
		if name == pluginManifestName {
			continue
		}
		if file.UncompressedSize64 > MaxPackageEntryBytes {
			return Manifest{}, fmt.Errorf("plugin file %q exceeds the size limit", name)
		}
		body, err := readZipEntry(file, MaxPackageEntryBytes)
		if err != nil {
			return Manifest{}, fmt.Errorf("read plugin file %q: %w", name, err)
		}
		packageFiles[name] = body
	}
	if PackageContentSHA256(packageFiles) != manifest.SHA256 {
		return Manifest{}, errors.New("plugin package sha256 does not match manifest")
	}
	if err := VerifyManifestSignature(manifest, trustedKeys); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, errors.New("archive entry exceeds the size limit")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit || uint64(len(data)) != file.UncompressedSize64 {
		return nil, errors.New("archive entry size does not match its directory record")
	}
	return data, nil
}


// PackageContentSHA256 computes the signed package digest independently of ZIP
// compression and entry order. File names and content lengths are framed to
// prevent ambiguous concatenations. manifest.json is excluded to avoid a
// self-referential digest.
func PackageContentSHA256(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		if name != pluginManifestName {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	hash := sha256.New()
	var length [8]byte
	for _, name := range names {
		binary.BigEndian.PutUint64(length[:], uint64(len(name)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(name))
		body := files[name]
		binary.BigEndian.PutUint64(length[:], uint64(len(body)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write(body)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
