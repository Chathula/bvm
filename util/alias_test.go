package util

import "testing"

func TestDefaultVersionRoundtrip(t *testing.T) {
	isolateEnv(t)

	def, err := DefaultVersion()
	if err != nil || def != "" {
		t.Fatalf("DefaultVersion() = (%q, %v), want empty", def, err)
	}

	if err := SetDefaultVersion("v1.4.0"); err != nil {
		t.Fatal(err)
	}
	def, err = DefaultVersion()
	if err != nil || def != "v1.4.0" {
		t.Fatalf("DefaultVersion() = (%q, %v), want v1.4.0", def, err)
	}
}
