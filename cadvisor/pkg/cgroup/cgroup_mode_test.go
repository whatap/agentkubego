package cgroup

import "testing"

func TestResolveCgroupModeUsesConfiguredValueWithoutDetection(t *testing.T) {
	called := false
	detected := func() string {
		called = true
		return "legacy"
	}

	got := resolveCgroupMode("unified", detected)

	if got != "unified" {
		t.Fatalf("resolveCgroupMode() = %q, want configured mode %q", got, "unified")
	}
	if called {
		t.Fatal("resolveCgroupMode() called detector despite configured mode")
	}
}
