package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
)

const (
	resultPrefix = "STRATAI_RESULT_JSON="
	buildID      = "golden-demo-semantic-runner@1"
)

var (
	aliasPattern    = regexp.MustCompile(`^demo-[0-9a-f]{20}$`)
	checksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type dependencies struct {
	parse  func(string) (*parserpkg.ParseDemoResult, error)
	export func(
		*models.DemoContext,
		string,
		string,
		string,
		models.CanonicalExportProvenance,
	) error
}

type commandResult struct {
	SchemaID    string `json:"schema_id"`
	Status      string `json:"status"`
	Stage       string `json:"stage"`
	SourceAlias string `json:"source_alias,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	BundleDir   string `json:"bundle_dir,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
}

type options struct {
	demoPath     string
	outputRoot   string
	sourceAlias  string
	checksum     string
	mapsRoot     string
	buildIDValue string
}

func main() {
	os.Exit(run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		dependencies{
			parse:  parserpkg.ParseDemoWithReplay,
			export: parserpkg.ExportMatchBundleWithProvenance,
		},
	))
}

func run(args []string, stdout, stderr io.Writer, deps dependencies) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		emitResult(stdout, commandResult{Status: "failed", Stage: "arguments", ErrorCode: "invalid_arguments"})
		return 2
	}
	if err := validateSourceIdentity(opts); err != nil {
		fmt.Fprintf(stderr, "source identity validation failed: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "source_identity", SourceAlias: opts.sourceAlias,
			ErrorCode: "source_identity_rejected",
		})
		return 3
	}

	if opts.mapsRoot != "" {
		if err := configureMapsRoot(opts.mapsRoot); err != nil {
			fmt.Fprintf(stderr, "maps root validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "configuration", SourceAlias: opts.sourceAlias,
				ErrorCode: "maps_root_rejected",
			})
			return 2
		}
	}

	if err := prepareOutputRoot(opts.outputRoot, opts.sourceAlias); err != nil {
		fmt.Fprintf(stderr, "output validation failed: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "publication", SourceAlias: opts.sourceAlias,
			ErrorCode: "output_rejected",
		})
		return 6
	}

	stagingRoot, err := os.MkdirTemp(opts.outputRoot, ".canonical-export-staging-")
	if err != nil {
		fmt.Fprintf(stderr, "create staging directory: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "publication", SourceAlias: opts.sourceAlias,
			ErrorCode: "staging_unavailable",
		})
		return 6
	}
	defer os.RemoveAll(stagingRoot)

	parsed, err := deps.parse(opts.demoPath)
	if err != nil {
		fmt.Fprintf(stderr, "parse demo: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "parse", SourceAlias: opts.sourceAlias,
			ErrorCode: "parser_rejected_demo",
		})
		return 4
	}
	if parsed == nil || parsed.Context == nil {
		fmt.Fprintln(stderr, "parse demo: parser returned no context")
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "parse", SourceAlias: opts.sourceAlias,
			ErrorCode: "parser_context_unavailable",
		})
		return 4
	}

	provenance := models.CanonicalExportProvenance{
		Source:          "demo",
		DemoChecksum:    opts.checksum,
		BuildIdentifier: opts.buildIDValue,
	}
	if err := deps.export(parsed.Context, opts.sourceAlias, stagingRoot, "", provenance); err != nil {
		fmt.Fprintf(stderr, "export canonical bundle: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "export", SourceAlias: opts.sourceAlias,
			ErrorCode: "canonical_export_rejected",
		})
		return 5
	}

	bundleName := "match_" + opts.sourceAlias
	stagedBundle := filepath.Join(stagingRoot, bundleName)
	finalBundle := filepath.Join(opts.outputRoot, bundleName)
	if err := os.Rename(stagedBundle, finalBundle); err != nil {
		fmt.Fprintf(stderr, "publish canonical bundle: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "publication", SourceAlias: opts.sourceAlias,
			ErrorCode: "atomic_publish_failed",
		})
		return 6
	}

	emitResult(stdout, commandResult{
		SchemaID: "stratai.golden_demo_export_result@1", Status: "exported", Stage: "complete",
		SourceAlias: opts.sourceAlias, SHA256: opts.checksum, BundleDir: bundleName,
	})
	return 0
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var opts options
	flags := flag.NewFlagSet("canonical-export", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.demoPath, "demo", "", "source .dem file")
	flags.StringVar(&opts.outputRoot, "output", "", "empty output root")
	flags.StringVar(&opts.sourceAlias, "alias", "", "opaque demo alias")
	flags.StringVar(&opts.checksum, "checksum", "", "expected lowercase SHA-256")
	flags.StringVar(&opts.mapsRoot, "maps-root", "", "optional CS2 geometry root")
	flags.StringVar(&opts.buildIDValue, "build-id", buildID, "deterministic build identifier")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("positional arguments are not allowed")
	}
	if strings.TrimSpace(opts.demoPath) == "" || strings.TrimSpace(opts.outputRoot) == "" ||
		strings.TrimSpace(opts.sourceAlias) == "" || strings.TrimSpace(opts.checksum) == "" {
		return options{}, errors.New("--demo, --output, --alias and --checksum are required")
	}
	if strings.TrimSpace(opts.buildIDValue) == "" {
		return options{}, errors.New("--build-id cannot be empty")
	}
	return opts, nil
}

func validateSourceIdentity(opts options) error {
	if !checksumPattern.MatchString(opts.checksum) {
		return errors.New("checksum must be 64 lowercase hexadecimal characters")
	}
	if !aliasPattern.MatchString(opts.sourceAlias) || opts.sourceAlias != "demo-"+opts.checksum[:20] {
		return errors.New("alias does not match checksum")
	}
	if !strings.EqualFold(filepath.Ext(opts.demoPath), ".dem") {
		return errors.New("source must use the .dem extension")
	}
	info, err := os.Lstat(opts.demoPath)
	if err != nil {
		return fmt.Errorf("inspect source: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("source must be a regular non-symlink file")
	}
	actual, err := sha256File(opts.demoPath)
	if err != nil {
		return fmt.Errorf("hash source: %w", err)
	}
	if actual != opts.checksum {
		return errors.New("source checksum mismatch")
	}
	return nil
}

func configureMapsRoot(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect maps root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("maps root must be a regular non-symlink directory")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve maps root: %w", err)
	}
	return os.Setenv("CS2_MAPS_DIR", absolute)
}

func prepareOutputRoot(root, sourceAlias string) error {
	info, err := os.Lstat(root)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(root, 0o750); err != nil {
			return err
		}
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("output root must be a regular non-symlink directory")
	}
	finalBundle := filepath.Join(root, "match_"+sourceAlias)
	if _, err := os.Lstat(finalBundle); err == nil {
		return errors.New("output bundle already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func sha256File(path string) (string, error) {
	handle, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer handle.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, handle); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func emitResult(writer io.Writer, result commandResult) {
	if result.SchemaID == "" {
		result.SchemaID = "stratai.golden_demo_export_result@1"
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return
	}
	fmt.Fprintf(writer, "%s%s\n", resultPrefix, payload)
}
