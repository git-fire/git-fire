package config_test

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/git-fire/git-fire/internal/config"
    "github.com/spf13/viper"
)

// TestSaveConfigRoundTripViaViper: SaveConfig writes integer cache_ttl;
// Viper should still parse it correctly.
func TestSaveConfigRoundTripViaViper(t *testing.T) {
    tmpDir := t.TempDir()
    cfgPath := filepath.Join(tmpDir, "config.toml")

    original := config.DefaultConfig()
    original.Global.DisableScan = true

    if err := config.SaveConfig(&original, cfgPath); err != nil {
        t.Fatalf("SaveConfig: %v", err)
    }

    data, err := os.ReadFile(cfgPath)
    if err != nil {
        t.Fatalf("ReadFile(%s): %v", cfgPath, err)
    }
    t.Logf("Saved config:\n%s", string(data))

    // Load via Viper (same as production)
    v := viper.New()
    v.SetConfigType("toml")
    v.SetConfigFile(cfgPath)
    if err := v.ReadInConfig(); err != nil {
        t.Fatalf("Viper ReadInConfig: %v", err)
    }

    var loaded config.Config
    if err := v.Unmarshal(&loaded); err != nil {
        t.Fatalf("Viper Unmarshal: %v", err)
    }

    t.Logf("cache_ttl raw: %v", v.Get("global.cache_ttl"))
    t.Logf("CacheTTL: %v", loaded.Global.CacheTTL)

    if !loaded.Global.DisableScan {
        t.Error("DisableScan: want true, got false")
    }
    if loaded.Global.CacheTTL != 24*time.Hour {
        t.Errorf("CacheTTL: want 24h (%v), got %v", 24*time.Hour, loaded.Global.CacheTTL)
    }
}
