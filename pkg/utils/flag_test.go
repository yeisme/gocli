package utils

import "testing"

func TestSetGlobalFlags(t *testing.T) {
	SetGlobalFlags(true, true, true, true)
	verbose, useColor, quiet, user := GetGlobalFlags()
	if !verbose || !useColor || !quiet || !user {
		t.Error("SetGlobalFlags did not set flags correctly")
	}
	if !IsVerbose() || !IsColor() || !IsQuiet() || !IsUser() {
		t.Error("Global flags getters did not return expected values")
	}
}

func TestSetConfigFile(t *testing.T) {
	SetConfigFile(".gocli.example.yaml")
	if configfile != ".gocli.example.yaml" {
		t.Error("SetConfigFile did not set configfile correctly")
	}
}
