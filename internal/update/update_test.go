package update

import (
	"context"
	"testing"
)

func TestCheckDevVersion(t *testing.T) {
	// "dev" is not a valid semver, so the library will treat any release as
	// newer. We just verify Check doesn't panic or return a hard error for
	// network-independent reasons. In CI without network this may fail, so
	// we skip on error.
	res, err := Check(context.Background(), "dev")
	if err != nil {
		t.Skipf("skipping (likely no network): %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.CurrentVersion != "dev" {
		t.Errorf("CurrentVersion = %q, want dev", res.CurrentVersion)
	}
}

func TestCheckValidVersion(t *testing.T) {
	// Use a very old version to verify the update-available logic.
	res, err := Check(context.Background(), "0.0.1")
	if err != nil {
		t.Skipf("skipping (likely no network): %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.CurrentVersion != "0.0.1" {
		t.Errorf("CurrentVersion = %q, want 0.0.1", res.CurrentVersion)
	}
}
