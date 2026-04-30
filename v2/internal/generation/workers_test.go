package generation_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

// ── ClampWorkers ──────────────────────────────────────────────────────────────

func TestClampWorkers(t *testing.T) {
	cases := []struct {
		n, fileCount, want int
	}{
		{0, 5, 1},
		{-3, 5, 1},
		{3, 5, 3},
		{10, 5, 5},
		{1, 1, 1},
		{1, 0, 1}, // fileCount=0 edge case; clamping to fileCount would give 0, but n<1 guard fires first
	}
	for _, tc := range cases {
		got := generation.ClampWorkers(tc.n, tc.fileCount)
		if got != tc.want {
			t.Errorf("ClampWorkers(%d, %d) = %d, want %d", tc.n, tc.fileCount, got, tc.want)
		}
	}
}

// ── concurrent Pipeline usage ─────────────────────────────────────────────────

// atomicDriver counts GenerateTestFile calls to verify each file is processed
// exactly once even under concurrent access.
type atomicDriver struct {
	fakeDriver
	calls atomic.Int32
}

func (d *atomicDriver) GenerateTestFile(a *domain.SourceAnalysis, o domain.GenerateOpts) (*domain.GeneratedFile, error) {
	d.calls.Add(1)
	return d.fakeDriver.GenerateTestFile(a, o)
}

func TestPipeline_ConcurrentPlanIsIdempotent(t *testing.T) {
	const fileCount = 12
	dir := t.TempDir()

	drv := &atomicDriver{fakeDriver: fakeDriver{lang: "go", testSuffix: "_test.go"}}
	pip := generation.NewPipeline(drv, nil)
	ex := &generation.Executor{}
	projCtx := &domain.ProjectContext{Root: dir, Language: "go"}
	opts := domain.GenerateOpts{}

	// Create source files.
	srcs := make([]string, fileCount)
	for i := range fileCount {
		p := filepath.Join(dir, filepath.FromSlash("src"), string(rune('a'+i))+".go")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package src\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		srcs[i] = p
	}

	// Run each file through plan+execute concurrently.
	sem := make(chan struct{}, 4) // 4 concurrent goroutines
	done := make(chan error, fileCount)
	for _, src := range srcs {
		src := src
		go func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			a, err := drv.AnalyzeFile(src, projCtx)
			if err != nil {
				done <- err
				return
			}
			plan, err := pip.Plan(context.Background(), a, opts)
			if err != nil {
				done <- err
				return
			}
			_, err = ex.Execute(plan)
			done <- err
		}()
	}

	for range fileCount {
		if err := <-done; err != nil {
			t.Errorf("concurrent run error: %v", err)
		}
	}

	if got := drv.calls.Load(); int(got) != fileCount {
		t.Errorf("GenerateTestFile called %d times, want %d", got, fileCount)
	}
}
