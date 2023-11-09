package util

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	t.Log("Testing Config Load")
	os.Setenv("PATCHTESTCONF_SIMPLE", "test")
	os.Setenv("PATCHTESTCONF_MAP_TEST", "abcde")
	config, err := LoadConfig("PATCHTESTCONF", nil)
	if err != nil {
		t.Error(err)
	}
	t.Logf("Config:\n%v", config.Sprint())
	if str, ok := config.Get("SiMPLe").(string); !ok || str != "test" {
		t.Error("Config is not loaded correctly (level 0)")
	}

	if confMap, ok := config.Get("MAP").(*Config); ok {
		if val, ok := confMap.GetString("test"); !ok || val != "abcde" {
			t.Error("Config is not loaded correctly (level 3)")
		}
	}

	if directTest, ok := config.GetString("MAP.TEST"); !ok || directTest != "abcde" {
		t.Error("Config is not loaded correctly (level 4)")
	}
}

func TestConfigWithFile(t *testing.T) {
	testValue := "abcdefg1234567!"
	t.Log("Testing Config Load with File")
	if err := os.WriteFile("test_conf.env", []byte(testValue), 0644); err != nil {
		t.Error("Failed to create test file")
		t.FailNow()
	}
	os.Setenv("PATCHTESTCONF_SIMPLE_FILE", "test_conf.env")
	// os.Setenv("PATCHTEST_MAP_FILE", "abcde")
	config, err := LoadConfig("PATCHTESTCONF", nil)
	if err != nil {
		t.Error(err)
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, ok := config.Get("SiMPLe").(string); !ok || val != testValue {
		t.Error("Config is not loaded correctly (level 1)")
	}

	if err := os.Remove("test_conf.env"); err != nil {
		t.Error("Failed to remove test file")
	}
}

func TestConfigWithInitialValue(t *testing.T) {
	initalValues := map[string]interface{}{
		"test":                "abcde",
		"nested.string":       "nestedTestValue",
		"example.test.number": 234567,
	}
	t.Log("Testing Config Load with Initial Values")
	config := NewConfig(initalValues, nil)
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, ok := config.GetString("test"); !ok || val != "abcde" {
		t.Error("Config is not loaded correctly (direct level 0)")
	}

	if nestedTest, ok := config.Get("nested").(*Config); ok {
		if val, ok := nestedTest.GetString("string"); !ok || val != "nestedTestValue" {
			t.Error("Config is not loaded correctly (level 1)")
		}
	} else {
		t.Error("Config has non-parsable sub config")
	}

	if val := config.Get("example.test.number").(int); val != 234567 {
		t.Error("Config is not loaded correctly (level 2)")
	}
}
