package system

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	t.Log("Testing Config Load")
	os.Setenv("PATCH_TESTSIMPLE", "test")
	os.Setenv("PATCH_TESTMAP_TEST", "abcde")
	config := LoadConfig("PATCH")
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, ok := config.Get("TEsTSiMPLe"); ok {
		if val.(string) != "test" {
			t.Error("Config is not loaded correctly (level 1)")
		}
	} else {
		t.Error("Config is not loaded correctly (level 0)")
	}

	if val, ok := config.Get("testMAP"); ok {
		if val, ok := val.(ConfigMap).Get("test"); ok {
			if val != "abcde" {
				t.Error("Config is not loaded correctly (level 3)")
			}
		} else {
			t.Error("Config is not loaded correctly (level 2)")
		}
	}
}
