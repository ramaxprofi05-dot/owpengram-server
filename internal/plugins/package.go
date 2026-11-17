package plugins

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
		if file.Mode()&io.ModeSymlink != 0 {
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

	entrypoint, ok := files[manifest.Entrypoint]
	if !ok {
		return Manifest{}, fmt.Errorf("plugin entrypoint %q is missing from archive", manifest.Entrypoint)
	}
	if entrypoint.UncompressedSize64 > MaxPackageEntryBytes {
		return Manifest{}, errors.New("plugin entrypoint exceeds the size limit")
	}
	entrypointBytes, err := readZipEntry(entrypoint, MaxPackageEntryBytes)
	if err != nil {
		return Manifest{}, fmt.Errorf("read plugin entrypoint: %w", err)
	}
	sum := sha256.Sum256(entrypointBytes)
	if hex.EncodeToString(sum[:]) != manifest.SHA256 {
		return Manifest{}, errors.New("plugin entrypoint sha256 does not match manifest")
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
