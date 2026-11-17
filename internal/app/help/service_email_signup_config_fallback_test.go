package help

import (
	"context"
	"encoding/json"
	"testing"

	"telesrv/internal/domain"
)

type fixedAppConfigStore struct {
	cfg   domain.AppConfig
	found bool
	err   error
}

func (s fixedAppConfigStore) GetAppConfig(context.Context, string) (domain.AppConfig, bool, error) {
	return s.cfg, s.found, s.err
}

func (s fixedAppConfigStore) UpsertAppConfig(context.Context, domain.AppConfig) error {
	return nil
}

func TestLoadAppConfigFallsBackWhenPersistedHashIsOlderThanRuntimeDefaults(t *testing.T) {
	ctx := context.Background()
	const staleHash = defaultAppConfigHash + 1 // newer than the static baseline, older than this runtime's email-signup config
	stale := domain.AppConfig{
		Client: tdesktopClient,
		Hash:   staleHash,
		JSON:   []byte(`{"legacy_seed_only":true}`),
	}
	service := NewService(fixedAppConfigStore{cfg: stale, found: true}, nil,
		WithEmailSignupEnable(true),
		WithEmailSignupPhonePrefixes([]string{"888", "777"}))

	got, _, err := service.GetAppConfig(ctx, 0, 0)
	if err != nil {
		t.Fatalf("GetAppConfig: %v", err)
	}
	if got.Hash <= staleHash {
		t.Fatalf("app config hash = %d, want runtime default hash newer than stale hash %d", got.Hash, staleHash)
	}
	if string(got.JSON) == string(stale.JSON) {
		t.Fatal("stale persisted app config masked runtime defaults")
	}
	if !containsJSONKey(got.JSON, "email_signup_enabled") {
		t.Fatalf("runtime email signup flag missing from fallback config: %s", got.JSON)
	}
	if !containsJSONKey(got.JSON, "email_signup_phone_prefixes") {
		t.Fatalf("runtime phone prefixes missing from fallback config: %s", got.JSON)
	}
}

func TestLoadAppConfigKeepsPersistedOperatorOverrideAtCurrentHash(t *testing.T) {
	ctx := context.Background()
	defaults := defaultAppConfig("",
		true, []string{"888"}, 0)
	override := domain.AppConfig{
		Client: tdesktopClient,
		Hash:   defaults.Hash + 10,
		JSON:   []byte(`{"operator_override":true}`),
	}
	service := NewService(fixedAppConfigStore{cfg: override, found: true}, nil,
		WithEmailSignupEnable(true),
		WithEmailSignupPhonePrefixes([]string{"888"}))

	got, _, err := service.GetAppConfig(ctx, 0, 0)
	if err != nil {
		t.Fatalf("GetAppConfig: %v", err)
	}
	if string(got.JSON) != string(override.JSON) {
		t.Fatalf("operator override was not preserved: got %s, want %s", got.JSON, override.JSON)
	}
}

func containsJSONKey(data []byte, key string) bool {
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return false
	}
	_, ok := values[key]
	return ok
}
