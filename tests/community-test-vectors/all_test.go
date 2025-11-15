package tests

import (
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

// readFileTrim reads a text file and trims trailing newlines.
func readFileTrim(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

// TestCommunityVectors validates the runtime against the public CBOR
// community test vectors stored under this directory.
func TestCommunityVectors(t *testing.T) {
	// This test file itself lives in tests/community-test-vectors.
	// Use the package directory as the root for walking vectors.
	root := "."
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		t.Fatalf("community vectors not present in %s; see tests/community-test-vectors/README.md", root)
	}

	var cases int
	type impl struct {
		name     string
		validate func([]byte) ([]byte, error)
		diag     func([]byte) (string, []byte, error)
	}

	impls := []impl{
		{
			name:     "runtime",
			validate: cbor.ValidateWellFormedBytes,
			diag:     cbor.DiagBytes,
		},
	}
	walkFn := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".cbor") {
			return nil
		}
		cases++
		caseName := strings.TrimPrefix(path, root+string(filepath.Separator))
		t.Run(caseName, func(t *testing.T) {
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			base := strings.TrimSuffix(path, ".cbor")
			diagPath := base + ".diag"
			hasDiag := false
			if _, err := os.Stat(diagPath); err == nil {
				hasDiag = true
			}

			for _, impl := range impls {
				impl := impl
				t.Run(impl.name, func(t *testing.T) {
					// Validate well-formed
					r, err := impl.validate(b)
					if err != nil {
						t.Fatalf("%s: well-formed failed for %s: %v", impl.name, path, err)
					}
					if len(r) != 0 {
						t.Fatalf("%s: leftover bytes after validation for %s: %d", impl.name, path, len(r))
					}
					// If .diag exists, compare
					if hasDiag {
						got, rest, err := impl.diag(b)
						if err != nil {
							t.Fatalf("%s: DiagBytes error for %s: %v", impl.name, path, err)
						}
						if len(rest) != 0 {
							t.Fatalf("%s: DiagBytes leftover for %s: %d", impl.name, path, len(rest))
						}
						want, err := readFileTrim(diagPath)
						if err != nil {
							t.Fatalf("read diag %s: %v", diagPath, err)
						}
						if got != want {
							t.Fatalf("%s: diag mismatch for %s:\n got: %q\nwant: %q", impl.name, path, got, want)
						}
					}
				})
			}
		})
		return nil
	}
	_ = filepath.Walk(root, walkFn)
	if cases == 0 {
		// Fallback: appendix_a.json at the root of the community vectors
		p := filepath.Join(root, "appendix_a.json")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Skip("no .cbor files found and appendix_a.json missing")
		}
		var vects []struct {
			Hex        string `json:"hex"`
			Diagnostic string `json:"diagnostic"`
		}
		if err := json.Unmarshal(b, &vects); err != nil {
			t.Fatalf("parse appendix_a.json: %v", err)
		}
		for i, v := range vects {
			if v.Hex == "" {
				continue
			}
			t.Run("appendix_a_"+strconv.Itoa(i), func(t *testing.T) {
				msg, err := hex.DecodeString(v.Hex)
				if err != nil {
					t.Fatalf("bad hex: %v", err)
				}
				for _, impl := range impls {
					impl := impl
					t.Run(impl.name, func(t *testing.T) {
						r, err := impl.validate(msg)
						if err != nil {
							t.Fatalf("%s: well-formed failed: %v", impl.name, err)
						}
						if len(r) != 0 {
							t.Fatalf("%s: leftover bytes: %d", impl.name, len(r))
						}
						if v.Diagnostic != "" {
							got, rest, err := impl.diag(msg)
							if err != nil {
								t.Fatalf("%s: diag error: %v", impl.name, err)
							}
							if len(rest) != 0 {
								t.Fatalf("%s: diag leftover: %d", impl.name, len(rest))
							}
							if got != v.Diagnostic {
								t.Fatalf("%s: diag mismatch: got %q want %q (hex %s)", impl.name, got, v.Diagnostic, v.Hex)
							}
						}
					})
				}
			})
		}
		if len(vects) == 0 {
			t.Skip("no vectors in appendix_a.json")
		}
	}
}
