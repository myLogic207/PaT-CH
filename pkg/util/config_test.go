package util

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	t.Log("Testing Config Load")
	os.Setenv("PATCHTESTCONF_SIMPLE", "test")
	os.Setenv("PATCHTESTCONF_MAP_TEST", "abcde")
	config := LoadConfig("PATCHTESTCONF", nil)
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, ok := config.Get("SiMPLe"); ok {
		if val.(string) != "test" {
			t.Error("Config is not loaded correctly (level 1)")
		}
	} else {
		t.Error("Config is not loaded correctly (level 0)")
	}

	if rawMap, ok := config.Get("MAP"); ok {
		if confMap, ok := rawMap.(*Config); ok {
			if val, ok := confMap.Get("test"); !ok || val != "abcde" {
				t.Error("Config is not loaded correctly (level 3)")
			}
		} else {
			t.Error("Config is not loaded correctly (level 2)")
		}
	}

	if directTest, ok := config.Get("MAP.TEST"); ok {
		if directTest != "abcde" {
			t.Error("Config is not loaded correctly (level 4)")
		}
	} else {
		t.Error("Config is not loaded correctly (level 3)")
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
	config := LoadConfig("PATCHTESTCONF", nil)
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

	if rawNestedTest, ok := config.Get("nested"); ok {
		if nestedTest, ok := rawNestedTest.(*Config); ok {
			if val, ok := nestedTest.GetString("string"); !ok || val != "nestedTestValue" {
				t.Error("Config is not loaded correctly (level 1)")
			}
		} else {
			t.Error("Config has non-parsable sub config")
		}
	} else {
		t.Error("Config is not loaded correctly (nested level 0)")
	}

	if val, ok := config.Get("example.test.number"); !ok {
		if val.(int) != 234567 {
			t.Error("Config is not loaded correctly (level 2)")
		}
	}

}
