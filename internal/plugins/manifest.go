package plugins

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaxManifestBytes = 64 << 10

type Target string

const (
	TargetAndroid Target = "android"
	TargetServer  Target = "server"
)

type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Target      Target   `json:"target"`
	Entrypoint  string   `json:"entrypoint"`
	Permissions []string `json:"permissions"`
	SHA256      string   `json:"sha256,omitempty"`
	SignerKeyID string   `json:"signer_key_id,omitempty"`
	Signature   string   `json:"signature,omitempty"`
}

var (
	pluginIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+){2,}$`)
	semverPattern   = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*))?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaxManifestBytes = 64 << 10

type Target string

const (
	TargetAndroid Target = "android"
	TargetServer  Target = "server"
)

type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Target      Target   `json:"target"`
	Entrypoint  string   `json:"entrypoint"`
	Permissions []string `json:"permissions"`
	SHA256      string   `json:"sha256,omitempty"`
	SignerKeyID string   `json:"signer_key_id,omitempty"`
	Signature   string   `json:"signature,omitempty"`
}

var (
	pluginIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+){2,}$`)
	)
	sha256Pattern   = regexp.MustCompile(`^[a-f0-9]{64}$`)
	keyIDPattern    = regexp.MustCompile(`^[a-zA-Z0-9._-]{8,96}$`)
	permissionSets  = map[Target]map[string]struct{}{
		TargetAndroid: setOf(
			"ui:chat-list", "ui:chat-screen", "ui:profile", "ui:profile-banner",
			"events:message-metadata", "events:profile-public",
			"storage:plugin", "network:https",
		),
		TargetServer: setOf(
			"server:message-metadata-hook", "server:profile-public-hook",
			"storage:plugin", "network:https",
		),
	}
)

func setOf(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

// ParseManifest accepts only the documented manifest schema. Unknown fields
// are rejected so a misspelled permission or policy field cannot be silently
// ignored during review.
func ParseManifest(data []byte) (Manifest, error) {
	if len(data) == 0 || len(data) > MaxManifestBytes {
		return Manifest{}, fmt.Errorf("plugin manifest must be between 1 and %d bytes", MaxManifestBytes)
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return Manifest{}, fmt.Errorf("plugin manifest contains ambiguous JSON: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode plugin manifest: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return Manifest{}, errors.New("plugin manifest must contain exactly one JSON object")
		}
		return Manifest{}, fmt.Errorf("trailing data after plugin manifest: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// rejectDuplicateJSONKeys prevents ambiguous manifests such as two different
// values for "permissions". encoding/json otherwise accepts duplicate object
// keys and silently keeps the later value.
func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("multiple top-level JSON values")
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate object key %q", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return errors.New("object is not properly closed")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return errors.New("array is not properly closed")
		}
	default:
		return fmt.Errorf("unexpected delimiter %q", delim)
	}
	return nil
}

func ValidateManifest(m Manifest) error {
	if !pluginIDPattern.MatchString(m.ID) || len(m.ID) > 128 {
		return errors.New("plugin id must be a reverse-domain identifier such as dev.example.weather")
	}
	if !validText(m.Name, 1, 64) {
		return errors.New("plugin name must be 1-64 characters without control characters")
	}
	if !semverPattern.MatchString(m.Version) {
		return errors.New("plugin version must use semantic versioning")
	}
	if !validText(m.Author, 1, 64) {
		return errors.New("plugin author must be 1-64 characters without control characters")
	}
	if !validText(m.Description, 1, 512) {
		return errors.New("plugin description must be 1-512 characters without control characters")
	}
	allowed, ok := permissionSets[m.Target]
	if !ok {
		return fmt.Errorf("unsupported plugin target %q", m.Target)
	}
	if !safeEntrypoint(m.Entrypoint, m.Target) {
		return errors.New("plugin entrypoint must be a safe relative file path for its target")
	}
	seen := make(map[string]struct{}, len(m.Permissions))
	for _, permission := range m.Permissions {
		if _, ok := allowed[permission]; !ok {
			return fmt.Errorf("permission %q is not allowed for target %q", permission, m.Target)
		}
		if _, duplicate := seen[permission]; duplicate {
			return fmt.Errorf("duplicate permission %q", permission)
		}
		seen[permission] = struct{}{}
	}
	if m.SHA256 != "" && !sha256Pattern.MatchString(m.SHA256) {
		return errors.New("sha256 must be a lowercase 64-character hexadecimal digest")
	}
	if m.SignerKeyID != "" && !keyIDPattern.MatchString(m.SignerKeyID) {
		return errors.New("signer_key_id has an invalid format")
	}
	if m.Signature != "" {
		raw, err := base64.StdEncoding.DecodeString(m.Signature)
		if err != nil || len(raw) != 64 {
			return errors.New("signature must be a base64-encoded Ed25519 signature")
		}
	}
	return nil
}

// SigningPayload returns the deterministic JSON payload covered by the Ed25519
// signature. The signature field is omitted; signer_key_id and sha256 remain
// covered so a signature cannot be moved to another publisher or package digest.
func SigningPayload(m Manifest) ([]byte, error) {
	if err := ValidateManifest(m); err != nil {
		return nil, err
	}
	m.Signature = ""
	return json.Marshal(m)
}

// VerifyManifestSignature checks an installable manifest against the explicit
// trusted publisher key set. A syntactically present signature is not proof
// of trust: callers must invoke this before accepting a package.
func VerifyManifestSignature(m Manifest, trustedKeys map[string]ed25519.PublicKey) error {
	if err := ValidateInstallableManifest(m); err != nil {
		return err
	}
	publicKey, ok := trustedKeys[m.SignerKeyID]
	if !ok {
		return fmt.Errorf("plugin signer %q is not trusted", m.SignerKeyID)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("trusted plugin signer %q has an invalid public key", m.SignerKeyID)
	}
	payload, err := SigningPayload(m)
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil {
		return errors.New("plugin signature is not valid base64")
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return errors.New("plugin signature verification failed")
	}
	return nil
}

// ValidateInstallableManifest adds the integrity requirements needed before
// a package can enter the plugin catalog. Signature verification against a
// trusted publisher key is performed by the installer, not by this parser.
func ValidateInstallableManifest(m Manifest) error {
	if err := ValidateManifest(m); err != nil {
		return err
	}
	if m.SHA256 == "" {
		return errors.New("installable plugin requires sha256")
	}
	if m.SignerKeyID == "" || m.Signature == "" {
		return errors.New("installable plugin requires a signer key id and signature")
	}
	return nil
}

// VerifyPluginPackage is the single pre-install trust gate. It validates
// archive paths and size limits, verifies the content digest over every
// package file, and checks the manifest signature against trusted publishers.
// The archive is inspected in place and no plugin code is extracted or run.
func VerifyPluginPackage(packageBytes []byte, trustedKeys map[string]ed25519.PublicKey) (Manifest, error) {
	return ValidatePackageArchive(packageBytes, trustedKeys)
}

func SafeArchivePath(name string) bool {
	if name == "" || strings.Contains(name, "\\") || strings.Contains(name, ":") || strings.ContainsRune(name, 0) || strings.HasPrefix(name, "/") {
		return false
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != name {
		return false
	}
	return true
}

func safeEntrypoint(name string, target Target) bool {
	if !SafeArchivePath(name) {
		return false
	}
	switch target {
	case TargetAndroid:
		return strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".mjs")
	case TargetServer:
		return strings.HasSuffix(name, ".wasm")
	default:
		return false
	}
}

func validText(value string, min, max int) bool {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) {
		return false
	}
	length := utf8.RuneCountInString(value)
	if length < min || length > max {
		return false
	}
	for _, r := range value {
		if r < 0x20 && r != '\t' {
			return false
		}
	}
	return true
}
