package util

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	t.Log("Testing Config Load")
	os.Setenv("PATCHTEST_SIMPLE", "test")
	os.Setenv("PATCHTEST_MAP_TEST", "abcde")
	config := LoadConfig("PATCHTEST")
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
		if val, ok := val.(*ConfigMap).Get("test"); ok {
			if val != "abcde" {
				t.Error("Config is not loaded correctly (level 3)")
			}
		} else {
			t.Error("Config is not loaded correctly (level 2)")
		}
	}
}

func TestConfigWithFile(t *testing.T) {
	testValue := "abcdefg1234567!"
	t.Log("Testing Config Load with File")
	if err := os.WriteFile("test_conf.env", []byte(testValue), 0644); err != nil {
		t.Error("Failed to create test file")
		t.FailNow()
	}
	os.Setenv("PATCHTEST_SIMPLE_FILE", "test_conf.env")
	// os.Setenv("PATCHTEST_MAP_FILE", "abcde")
	config := LoadConfig("PATCHTEST")
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, ok := config.Get("SiMPLe"); ok {
		if val.(string) != testValue {
			t.Error("Config is not loaded correctly (level 1)")
		}
	} else {
		t.Error("Config is not loaded correctly (level 0)")
	}

	if err := os.Remove("test_conf.env"); err != nil {
		t.Error("Failed to remove test file")
	}
}
