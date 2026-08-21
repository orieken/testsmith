package python

import (
	"testing"
)

// backfill / AC: splitPath returns all path components as a slice
func TestSplitPath_MultiSegment(t *testing.T) {
	t.Parallel()
	got := splitPath("src/services/payment.py")
	want := []string{"src", "services", "payment.py"}
	if len(got) != len(want) {
		t.Fatalf("splitPath(\"src/services/payment.py\") = %v (len %d), want %v (len %d)",
			got, len(got), want, len(want))
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("splitPath[%d] = %q, want %q", i, g, want[i])
		}
	}
}

// backfill / AC: splitPath returns single element for a plain filename
func TestSplitPath_SingleSegment(t *testing.T) {
	t.Parallel()
	got := splitPath("myapp")
	if len(got) != 1 || got[0] != "myapp" {
		t.Errorf("splitPath(\"myapp\") = %v, want [\"myapp\"]", got)
	}
}

// backfill / AC: splitPath returns empty slice for "." (current directory)
func TestSplitPath_Dot(t *testing.T) {
	t.Parallel()
	got := splitPath(".")
	if len(got) != 0 {
		t.Errorf("splitPath(\".\") = %v, want []", got)
	}
}

// backfill / AC: splitPath returns empty slice for empty string
func TestSplitPath_Empty(t *testing.T) {
	t.Parallel()
	got := splitPath("")
	if len(got) != 0 {
		t.Errorf("splitPath(\"\") = %v, want []", got)
	}
}

// backfill / AC: splitPath handles three-level path correctly
func TestSplitPath_ThreeLevels(t *testing.T) {
	t.Parallel()
	got := splitPath("a/b/c")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitPath(\"a/b/c\") = %v, want %v", got, want)
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("splitPath[%d] = %q, want %q", i, g, want[i])
		}
	}
}

// backfill / AC: splitPath handles src-layout prefix used by scanPackages
func TestSplitPath_SrcLayout(t *testing.T) {
	t.Parallel()
	got := splitPath("src/mypackage")
	want := []string{"src", "mypackage"}
	if len(got) != len(want) {
		t.Fatalf("splitPath(\"src/mypackage\") = %v, want %v", got, want)
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("splitPath[%d] = %q, want %q", i, g, want[i])
		}
	}
}
