package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/delaneyj/cbor/cborgen/core"
)

// CLI defines the cborgen command-line interface.
//
// We deliberately keep it minimal:
//   - input: Go file or directory
//   - output: override for the generated file (file mode only)
//   - verbose: turn on diagnostic logging
//
// In directory mode, each source file gets its own
// "*_cbor.go" companion file and the --output flag is rejected.
type CLI struct {
	Input   string   `short:"i" help:"Input Go file or directory" default:"${env:GOFILE}"`
	Output  string   `short:"o" help:"Output file (file input only; defaults to {input}_cbor.go)"`
	Structs []string `short:"s" help:"Only generate for these struct types (may be repeated)"`
	Verbose bool     `short:"v" help:"Enable verbose diagnostics"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("cborgen"),
		kong.Description("Generate CBOR encoders/decoders with Safe and Trusted variants."),
	)

	if err := run(&cli); err != nil {
		ctx.FatalIfErrorf(err)
	}
}

func run(cli *CLI) error {
	input := strings.TrimSpace(cli.Input)
	if input == "" {
		return errors.New("no input specified (use --input or set GOFILE)")
	}

	info, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("stat input: %w", err)
	}

	if info.IsDir() {
		if cli.Output != "" {
			return errors.New("--output is not allowed when input is a directory")
		}
		return runForDir(input, cli.Verbose, cli.Structs)
	}

	// Single-file mode.
	out := cli.Output
	if strings.TrimSpace(out) == "" {
		out = defaultOutputPath(input)
	}
	return generateForFile(input, out, cli.Verbose, cli.Structs)
}

// runForDir walks a directory and generates a companion
// "*_cbor.go" file for each eligible Go source file.
func runForDir(dir string, verbose bool, structs []string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_cbor.go") {
			continue
		}

		inPath := filepath.Join(dir, name)
		info, err := entry.Info()
		if err != nil {
			// If we can't stat a file, treat it as fatal
			// to avoid silently skipping sources.
			return fmt.Errorf("stat %q: %w", inPath, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}

		outPath := defaultOutputPath(inPath)
		if err := generateForFile(inPath, outPath, verbose, structs); err != nil {
			return err
		}
	}

	return nil
}

// defaultOutputPath derives the "*_cbor.go" filename for
// a given input Go file path.
func defaultOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	if !strings.HasSuffix(base, ".go") {
		// Should not normally happen, but fall back sensibly.
		return filepath.Join(dir, base+"_cbor.go")
	}
	name := strings.TrimSuffix(base, ".go") + "_cbor.go"
	return filepath.Join(dir, name)
}

func generateForFile(inputPath, outputPath string, verbose bool, structs []string) error {
	return core.Run(inputPath, outputPath, core.Options{Verbose: verbose, Structs: structs})
}
