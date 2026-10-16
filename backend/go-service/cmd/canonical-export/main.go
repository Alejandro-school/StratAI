package main

import (
	"compress/gzip"
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
	"sort"
	"strings"
	"time"

	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
	"cs2-demo-service/pkg/cover"
	"cs2-demo-service/pkg/exitroute"
	"cs2-demo-service/pkg/exposure"
	mapspkg "cs2-demo-service/pkg/maps"
	"cs2-demo-service/pkg/navigation"
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
	parse                         func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomy                  func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyV2                func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomySources           func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyRules             func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomySessions          func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyManifest          func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyTrace             func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyIdentity          func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyOwnership         func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyEntries           func(string) (*parserpkg.ParseDemoResult, error)
	parseEconomyCashflow          func(string) (*parserpkg.ParseDemoResult, error)
	parseSupport                  func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportPhysical          func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportObjective         func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportC4                func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportAttemptOrigin     func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportObjectivePosition func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportNativeMotion      func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportVisualContact     func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportWeaponTiming      func(string) (*parserpkg.ParseDemoResult, error)
	parseSupportObjectiveClock    func(string) (*parserpkg.ParseDemoResult, error)
	export                        func(
		*models.DemoContext,
		string,
		string,
		string,
		models.CanonicalExportProvenance,
	) error
	navigationSmoke func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		int,
	) (parserpkg.ObservedNavigationSmokeReport, error)
	visualContactSmoke func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		string,
		int,
	) (parserpkg.ObservedVisualContactSmokeReport, error)
	navigationFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedNavigationFullDemoReport, error)
	visualContactFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedVisualContactFullDemoReport, error)
	weaponStateFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedWeaponStateFullDemoReport, error)
	utilityInitialVelocityFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedUtilityInitialVelocityFullDemoReport, error)
	lastKnownPositionFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedLastKnownPositionFullDemoReport, error)
	exposureAngleFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedExposureAngleFullDemoReport, error)
	coverFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedCoverFullDemoReport, error)
	exitRouteFullDemo func(
		*parserpkg.ParseDemoResult,
		string,
		string,
		string,
		string,
		io.Writer,
	) (parserpkg.ObservedExitRouteFullDemoReport, error)
	validateNavigationFullDemo             func(parserpkg.ObservedNavigationFullDemoReport) error
	validateVisualContactFullDemo          func(parserpkg.ObservedVisualContactFullDemoReport) error
	validateWeaponStateFullDemo            func(parserpkg.ObservedWeaponStateFullDemoReport) error
	validateUtilityInitialVelocityFullDemo func(parserpkg.ObservedUtilityInitialVelocityFullDemoReport) error
	validateLastKnownPositionFullDemo      func(parserpkg.ObservedLastKnownPositionFullDemoReport) error
	validateExposureAngleFullDemo          func(parserpkg.ObservedExposureAngleFullDemoReport) error
	validateCoverFullDemo                  func(parserpkg.ObservedCoverFullDemoReport) error
	validateExitRouteFullDemo              func(parserpkg.ObservedExitRouteFullDemoReport) error
}

type commandResult struct {
	EconomyCandidateManifest             string `json:"economy_candidate_manifest,omitempty"`
	SchemaID                             string `json:"schema_id"`
	Status                               string `json:"status"`
	Stage                                string `json:"stage"`
	SourceAlias                          string `json:"source_alias,omitempty"`
	SHA256                               string `json:"sha256,omitempty"`
	BundleDir                            string `json:"bundle_dir,omitempty"`
	SmokeReport                          string `json:"smoke_report,omitempty"`
	VisualContactSmokeReport             string `json:"visual_contact_smoke_report,omitempty"`
	NavigationFullDemoReport             string `json:"navigation_full_demo_report,omitempty"`
	NavigationFullDemoRows               string `json:"navigation_full_demo_rows,omitempty"`
	VisualContactFullDemoReport          string `json:"visual_contact_full_demo_report,omitempty"`
	VisualContactFullDemoAudit           string `json:"visual_contact_full_demo_audit,omitempty"`
	WeaponStateFullDemoReport            string `json:"weapon_state_full_demo_report,omitempty"`
	WeaponStateFullDemoRows              string `json:"weapon_state_full_demo_rows,omitempty"`
	UtilityInitialVelocityFullDemoReport string `json:"utility_initial_velocity_full_demo_report,omitempty"`
	UtilityInitialVelocityFullDemoRows   string `json:"utility_initial_velocity_full_demo_rows,omitempty"`
	LastKnownPositionFullDemoReport      string `json:"last_known_position_full_demo_report,omitempty"`
	LastKnownPositionFullDemoRows        string `json:"last_known_position_full_demo_rows,omitempty"`
	ExposureAngleFullDemoReport          string `json:"exposure_angle_full_demo_report,omitempty"`
	ExposureAngleFullDemoRows            string `json:"exposure_angle_full_demo_rows,omitempty"`
	CoverFullDemoReport                  string `json:"cover_full_demo_report,omitempty"`
	CoverFullDemoRows                    string `json:"cover_full_demo_rows,omitempty"`
	ExitRouteFullDemoReport              string `json:"exit_route_full_demo_report,omitempty"`
	ExitRouteFullDemoRows                string `json:"exit_route_full_demo_rows,omitempty"`
	ErrorCode                            string `json:"error_code,omitempty"`
}

type options struct {
	demoPath                                 string
	outputRoot                               string
	sourceAlias                              string
	checksum                                 string
	mapsRoot                                 string
	exposurePackagesRoot                     string
	coverPackagesRoot                        string
	exitRoutePackagesRoot                    string
	buildIDValue                             string
	navigationSmoke                          bool
	aiCoachSmoke                             bool
	aiCoachFullDemoValidation                bool
	aiCoachExposureValidation                bool
	aiCoachCoverValidation                   bool
	aiCoachExitRouteValidation               bool
	aiCoachSpatialCandidate                  bool
	aiCoachEconomyCandidate                  bool
	aiCoachSupportCandidate                  bool
	aiCoachSupportPhysicalCandidate          bool
	aiCoachSupportObjectiveCandidate         bool
	aiCoachSupportC4Candidate                bool
	aiCoachSupportObjectiveClockCandidate    bool
	aiCoachSupportAttemptOriginCandidate     bool
	aiCoachSupportObjectivePositionCandidate bool
	aiCoachSupportNativeMotionCandidate      bool
	aiCoachSupportVisualContactCandidate     bool
	aiCoachSupportWeaponTimingCandidate      bool
	economyCandidateVersion                  int
	navigationSmokeQueries                   int
	visualContactSmokeFrames                 int
}

func main() {
	os.Exit(run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		dependencies{
			parse:                                  parserpkg.ParseDemoWithReplay,
			parseEconomy:                           parserpkg.ParseDemoWithEconomyObservation,
			parseEconomyV2:                         parserpkg.ParseDemoWithEconomyInitialFrontiers,
			parseEconomySources:                    parserpkg.ParseDemoWithEconomyNativeSources,
			parseEconomyRules:                      parserpkg.ParseDemoWithEconomyRuleContext,
			parseEconomySessions:                   parserpkg.ParseDemoWithEconomySessionPayloads,
			parseEconomyManifest:                   parserpkg.ParseDemoWithEconomyManifest,
			parseEconomyTrace:                      parserpkg.ParseDemoWithEconomyTransactionTrace,
			parseEconomyIdentity:                   parserpkg.ParseDemoWithEconomyNativeIdentity,
			parseEconomyOwnership:                  parserpkg.ParseDemoWithEconomyOwnership,
			parseEconomyEntries:                    parserpkg.ParseDemoWithEconomyEntries,
			parseEconomyCashflow:                   parserpkg.ParseDemoWithEconomyCashflow,
			parseSupport:                           parserpkg.ParseDemoWithSupportObservations,
			parseSupportPhysical:                   parserpkg.ParseDemoWithSupportPhysical,
			parseSupportObjective:                  parserpkg.ParseDemoWithSupportObjective,
			parseSupportC4:                         parserpkg.ParseDemoWithSupportC4,
			parseSupportObjectiveClock:             parserpkg.ParseDemoWithSupportObjectiveClock,
			parseSupportAttemptOrigin:              parserpkg.ParseDemoWithSupportAttemptOrigin,
			parseSupportObjectivePosition:          parserpkg.ParseDemoWithSupportObjectivePosition,
			parseSupportNativeMotion:               parserpkg.ParseDemoWithSupportNativeMotion,
			parseSupportVisualContact:              parserpkg.ParseDemoWithSupportVisualContact,
			parseSupportWeaponTiming:               parserpkg.ParseDemoWithSupportWeaponTiming,
			export:                                 parserpkg.ExportMatchBundleWithProvenance,
			navigationSmoke:                        buildNavigationSmokeReport,
			visualContactSmoke:                     buildVisualContactSmokeReport,
			navigationFullDemo:                     buildNavigationFullDemoReport,
			visualContactFullDemo:                  buildVisualContactFullDemoReport,
			weaponStateFullDemo:                    buildObservedWeaponStateFullDemoReport,
			utilityInitialVelocityFullDemo:         buildObservedUtilityInitialVelocityFullDemoReport,
			lastKnownPositionFullDemo:              buildObservedLastKnownPositionFullDemoReport,
			exposureAngleFullDemo:                  buildObservedExposureAngleFullDemoReport,
			coverFullDemo:                          buildObservedCoverFullDemoReport,
			exitRouteFullDemo:                      buildObservedExitRouteFullDemoReport,
			validateNavigationFullDemo:             parserpkg.ValidateObservedNavigationFullDemoReport,
			validateVisualContactFullDemo:          parserpkg.ValidateObservedVisualContactFullDemoReport,
			validateWeaponStateFullDemo:            parserpkg.ValidateObservedWeaponStateFullDemoReport,
			validateUtilityInitialVelocityFullDemo: parserpkg.ValidateObservedUtilityInitialVelocityFullDemoReport,
			validateLastKnownPositionFullDemo:      parserpkg.ValidateObservedLastKnownPositionFullDemoReport,
			validateExposureAngleFullDemo:          parserpkg.ValidateObservedExposureAngleFullDemoReport,
			validateCoverFullDemo:                  parserpkg.ValidateObservedCoverFullDemoReport,
			validateExitRouteFullDemo:              parserpkg.ValidateObservedExitRouteFullDemoReport,
		},
	))
}

func run(args []string, stdout, stderr io.Writer, deps dependencies) int {
	startedAt := time.Now()
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
	navigationSmokeEnabled := opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation
	smokeEnabled := navigationSmokeEnabled || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate || opts.aiCoachSupportCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate
	if smokeEnabled {
		if err := validateNavigationSmokeOutputRoot(opts.outputRoot); err != nil {
			fmt.Fprintf(stderr, "AI Coach smoke output validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "publication", SourceAlias: opts.sourceAlias,
				ErrorCode: "smoke_output_rejected",
			})
			return 6
		}
	}

	publicationName := "match_" + opts.sourceAlias
	if smokeEnabled {
		publicationName = "smoke_run_" + opts.sourceAlias
	}
	if err := prepareOutputRoot(opts.outputRoot, publicationName, smokeEnabled); err != nil {
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
	exportRoot := stagingRoot
	if smokeEnabled {
		exportRoot = filepath.Join(stagingRoot, publicationName)
		if err := os.MkdirAll(exportRoot, 0o750); err != nil {
			fmt.Fprintf(stderr, "create smoke publication directory: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "publication", SourceAlias: opts.sourceAlias,
				ErrorCode: "staging_unavailable",
			})
			return 6
		}
	}

	parse := deps.parse
	if opts.aiCoachSupportCandidate {
		parse = deps.parseSupport
	}
	if opts.aiCoachSupportPhysicalCandidate {
		parse = deps.parseSupportPhysical
	}
	if opts.aiCoachSupportObjectiveCandidate {
		parse = deps.parseSupportObjective
	}
	if opts.aiCoachSupportVisualContactCandidate {
		parse = deps.parseSupportVisualContact
	} else if opts.aiCoachSupportWeaponTimingCandidate {
		parse = deps.parseSupportWeaponTiming
	} else if opts.aiCoachSupportNativeMotionCandidate {
		parse = deps.parseSupportNativeMotion
	} else if opts.aiCoachSupportObjectivePositionCandidate {
		parse = deps.parseSupportObjectivePosition
	} else if opts.aiCoachSupportAttemptOriginCandidate {
		parse = deps.parseSupportAttemptOrigin
	} else if opts.aiCoachSupportObjectiveClockCandidate {
		parse = deps.parseSupportObjectiveClock
	} else if opts.aiCoachSupportC4Candidate {
		parse = deps.parseSupportC4
	}
	if opts.aiCoachEconomyCandidate {
		parse = deps.parseEconomy
		if opts.economyCandidateVersion == 2 {
			parse = deps.parseEconomyV2
		}
		if opts.economyCandidateVersion == 3 {
			parse = deps.parseEconomySources
		}
		if opts.economyCandidateVersion == 4 {
			parse = deps.parseEconomyRules
		}
		if opts.economyCandidateVersion == 5 {
			parse = deps.parseEconomySessions
		}
		if opts.economyCandidateVersion == 6 {
			parse = deps.parseEconomyManifest
		}
		if opts.economyCandidateVersion == 7 {
			parse = deps.parseEconomyTrace
		}
		if opts.economyCandidateVersion == 8 {
			parse = deps.parseEconomyIdentity
		}
		if opts.economyCandidateVersion == 9 {
			parse = deps.parseEconomyOwnership
		}
		if opts.economyCandidateVersion == 10 {
			parse = deps.parseEconomyEntries
		}
		if opts.economyCandidateVersion == 11 {
			parse = deps.parseEconomyCashflow
		}
	}
	if parse == nil {
		fmt.Fprintln(stderr, "requested parser is unavailable")
		return 4
	}
	parsed, err := parse(opts.demoPath)
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

	var navigationSmokeReport parserpkg.ObservedNavigationSmokeReport
	if navigationSmokeEnabled {
		if parsed.ReplayData == nil {
			fmt.Fprintln(stderr, "navigation smoke: parser returned no replay data")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_smoke_replay_unavailable",
			})
			return 7
		}
		if deps.navigationSmoke == nil {
			fmt.Fprintln(stderr, "navigation smoke: builder is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_smoke_builder_unavailable",
			})
			return 7
		}
		navigationSmokeReport, err = deps.navigationSmoke(
			parsed,
			opts.sourceAlias,
			opts.checksum,
			opts.navigationSmokeQueries,
		)
		if err != nil {
			fmt.Fprintf(stderr, "navigation smoke validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_smoke_rejected",
			})
			return 7
		}
	}

	var visualContactSmokeReport parserpkg.ObservedVisualContactSmokeReport
	if opts.aiCoachSmoke || opts.aiCoachFullDemoValidation {
		if deps.visualContactSmoke == nil {
			fmt.Fprintln(stderr, "visual contact smoke: builder is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_smoke_builder_unavailable",
			})
			return 8
		}
		visualContactSmokeReport, err = deps.visualContactSmoke(
			parsed,
			opts.sourceAlias,
			opts.checksum,
			opts.mapsRoot,
			opts.visualContactSmokeFrames,
		)
		if err != nil {
			fmt.Fprintf(stderr, "visual contact smoke validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_smoke_rejected",
			})
			return 8
		}
	}

	var navigationFullDemoReport parserpkg.ObservedNavigationFullDemoReport
	var visualContactFullDemoReport parserpkg.ObservedVisualContactFullDemoReport
	var weaponStateFullDemoReport parserpkg.ObservedWeaponStateFullDemoReport
	var utilityInitialVelocityFullDemoReport parserpkg.ObservedUtilityInitialVelocityFullDemoReport
	var lastKnownPositionFullDemoReport parserpkg.ObservedLastKnownPositionFullDemoReport
	var exposureAngleFullDemoReport parserpkg.ObservedExposureAngleFullDemoReport
	var coverFullDemoReport parserpkg.ObservedCoverFullDemoReport
	var exitRouteFullDemoReport parserpkg.ObservedExitRouteFullDemoReport
	if opts.aiCoachFullDemoValidation {
		if deps.navigationFullDemo == nil || deps.validateNavigationFullDemo == nil {
			fmt.Fprintln(stderr, "full-demo navigation: builder or validator is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_full_demo_builder_unavailable",
			})
			return 9
		}
		navigationStartedAt := time.Now()
		navigationRowsPath := filepath.Join(exportRoot, parserpkg.ObservedNavigationFullDemoRowsFilename)
		navigationBytes, navigationSHA256, buildErr := writeDeterministicGzipArtifact(
			navigationRowsPath,
			func(writer io.Writer) error {
				var err error
				navigationFullDemoReport, err = deps.navigationFullDemo(
					parsed,
					opts.sourceAlias,
					opts.checksum,
					writer,
				)
				return err
			},
		)
		if buildErr != nil {
			fmt.Fprintf(stderr, "full-demo navigation validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_full_demo_rejected",
			})
			return 9
		}
		navigationFullDemoReport.BuildIdentifier = opts.buildIDValue
		navigationDurationMS := time.Since(navigationStartedAt).Milliseconds()
		navigationFullDemoReport.NavigationEvaluationDurationMS = navigationDurationMS
		navigationFullDemoReport.ProcessingDurationMS = navigationDurationMS
		navigationFullDemoReport.RowsArtifact.Path = parserpkg.ObservedNavigationFullDemoRowsFilename
		navigationFullDemoReport.RowsArtifact.Bytes = navigationBytes
		navigationFullDemoReport.RowsArtifact.SHA256 = navigationSHA256
		if err := deps.validateNavigationFullDemo(navigationFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "full-demo navigation report validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_full_demo_rejected",
			})
			return 9
		}

		if deps.visualContactFullDemo == nil || deps.validateVisualContactFullDemo == nil {
			fmt.Fprintln(stderr, "full-demo visual contact: builder or validator is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_full_demo_builder_unavailable",
			})
			return 10
		}
		visualStartedAt := time.Now()
		visualAuditPath := filepath.Join(exportRoot, parserpkg.ObservedVisualContactFullDemoAuditFilename)
		visualBytes, visualSHA256, buildErr := writeDeterministicGzipArtifact(
			visualAuditPath,
			func(writer io.Writer) error {
				var err error
				visualContactFullDemoReport, err = deps.visualContactFullDemo(
					parsed,
					opts.sourceAlias,
					opts.checksum,
					opts.mapsRoot,
					writer,
				)
				return err
			},
		)
		if buildErr != nil {
			fmt.Fprintf(stderr, "full-demo visual-contact validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_full_demo_rejected",
			})
			return 10
		}
		visualContactFullDemoReport.BuildIdentifier = opts.buildIDValue
		visualContactFullDemoReport.ProcessingDurationMS = time.Since(visualStartedAt).Milliseconds()
		visualContactFullDemoReport.Audit.GzipRelativePath = parserpkg.ObservedVisualContactFullDemoAuditFilename
		visualContactFullDemoReport.Audit.GzipBytes = visualBytes
		visualContactFullDemoReport.Audit.GzipSHA256 = visualSHA256
		if err := deps.validateVisualContactFullDemo(visualContactFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "full-demo visual-contact report validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_full_demo_rejected",
			})
			return 10
		}

		if deps.weaponStateFullDemo == nil || deps.validateWeaponStateFullDemo == nil {
			fmt.Fprintln(stderr, "full-demo weapon state: builder or validator is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "weapon_state_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "weapon_state_full_demo_builder_unavailable",
			})
			return 11
		}
		weaponRowsPath := filepath.Join(exportRoot, parserpkg.ObservedWeaponStateFullDemoRowsFilename)
		weaponBytes, weaponSHA256, buildErr := writeDeterministicGzipArtifact(
			weaponRowsPath,
			func(writer io.Writer) error {
				var err error
				weaponStateFullDemoReport, err = deps.weaponStateFullDemo(
					parsed,
					opts.sourceAlias,
					opts.checksum,
					writer,
				)
				return err
			},
		)
		if buildErr != nil {
			fmt.Fprintf(stderr, "full-demo weapon-state validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "weapon_state_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "weapon_state_full_demo_rejected",
			})
			return 11
		}
		weaponStateFullDemoReport.BuildIdentifier = opts.buildIDValue
		weaponStateFullDemoReport.RowsArtifact.Path = parserpkg.ObservedWeaponStateFullDemoRowsFilename
		weaponStateFullDemoReport.RowsArtifact.Bytes = weaponBytes
		weaponStateFullDemoReport.RowsArtifact.SHA256 = weaponSHA256
		if err := deps.validateWeaponStateFullDemo(weaponStateFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "full-demo weapon-state report validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "weapon_state_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "weapon_state_full_demo_rejected",
			})
			return 11
		}

		if deps.utilityInitialVelocityFullDemo == nil || deps.validateUtilityInitialVelocityFullDemo == nil {
			fmt.Fprintln(stderr, "full-demo utility initial velocity: builder or validator is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "utility_initial_velocity_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "utility_initial_velocity_full_demo_builder_unavailable",
			})
			return 12
		}
		utilityVelocityRowsPath := filepath.Join(exportRoot, parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename)
		utilityVelocityBytes, utilityVelocitySHA256, buildErr := writeDeterministicGzipArtifact(
			utilityVelocityRowsPath,
			func(writer io.Writer) error {
				var err error
				utilityInitialVelocityFullDemoReport, err = deps.utilityInitialVelocityFullDemo(
					parsed,
					opts.sourceAlias,
					opts.checksum,
					writer,
				)
				return err
			},
		)
		if buildErr != nil {
			fmt.Fprintf(stderr, "full-demo utility initial-velocity validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "utility_initial_velocity_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "utility_initial_velocity_full_demo_rejected",
			})
			return 12
		}
		utilityInitialVelocityFullDemoReport.BuildIdentifier = opts.buildIDValue
		utilityInitialVelocityFullDemoReport.RowsArtifact.Path = parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename
		utilityInitialVelocityFullDemoReport.RowsArtifact.Bytes = utilityVelocityBytes
		utilityInitialVelocityFullDemoReport.RowsArtifact.SHA256 = utilityVelocitySHA256
		if err := deps.validateUtilityInitialVelocityFullDemo(utilityInitialVelocityFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "full-demo utility initial-velocity report validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "utility_initial_velocity_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "utility_initial_velocity_full_demo_rejected",
			})
			return 12
		}

		if deps.lastKnownPositionFullDemo == nil || deps.validateLastKnownPositionFullDemo == nil {
			fmt.Fprintln(stderr, "full-demo last-known position: builder or validator is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "last_known_position_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "last_known_position_full_demo_builder_unavailable",
			})
			return 13
		}
		lastKnownRowsPath := filepath.Join(exportRoot, parserpkg.ObservedLastKnownPositionFullDemoRowsFilename)
		lastKnownBytes, lastKnownSHA256, buildErr := writeDeterministicGzipArtifact(
			lastKnownRowsPath,
			func(writer io.Writer) error {
				var err error
				lastKnownPositionFullDemoReport, err = deps.lastKnownPositionFullDemo(
					parsed,
					opts.sourceAlias,
					opts.checksum,
					writer,
				)
				return err
			},
		)
		if buildErr != nil {
			fmt.Fprintf(stderr, "full-demo last-known position validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "last_known_position_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "last_known_position_full_demo_rejected",
			})
			return 13
		}
		lastKnownPositionFullDemoReport.BuildIdentifier = opts.buildIDValue
		lastKnownPositionFullDemoReport.RowsArtifact.Path = parserpkg.ObservedLastKnownPositionFullDemoRowsFilename
		lastKnownPositionFullDemoReport.RowsArtifact.Bytes = lastKnownBytes
		lastKnownPositionFullDemoReport.RowsArtifact.SHA256 = lastKnownSHA256
		if err := deps.validateLastKnownPositionFullDemo(lastKnownPositionFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "full-demo last-known position report validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "last_known_position_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "last_known_position_full_demo_rejected",
			})
			return 13
		}
	}
	if opts.aiCoachExposureValidation {
		if deps.exposureAngleFullDemo == nil || deps.validateExposureAngleFullDemo == nil {
			fmt.Fprintln(stderr, "exposure-angle sidecar: builder or validator is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "exposure_angle_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "exposure_angle_full_demo_builder_unavailable",
			})
			return 14
		}
		exposureRowsPath := filepath.Join(exportRoot, parserpkg.ObservedExposureAngleRowsFilename)
		exposureBytes, exposureSHA256, buildErr := writeDeterministicGzipArtifact(
			exposureRowsPath,
			func(writer io.Writer) error {
				var err error
				exposureAngleFullDemoReport, err = deps.exposureAngleFullDemo(
					parsed,
					opts.sourceAlias,
					opts.checksum,
					opts.mapsRoot,
					opts.exposurePackagesRoot,
					writer,
				)
				return err
			},
		)
		if buildErr != nil {
			fmt.Fprintf(stderr, "exposure-angle sidecar validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "exposure_angle_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "exposure_angle_full_demo_rejected",
			})
			return 14
		}
		exposureAngleFullDemoReport.BuildIdentifier = opts.buildIDValue
		exposureAngleFullDemoReport.RowsArtifact.Path = parserpkg.ObservedExposureAngleRowsFilename
		exposureAngleFullDemoReport.RowsArtifact.Bytes = exposureBytes
		exposureAngleFullDemoReport.RowsArtifact.SHA256 = exposureSHA256
		if err := deps.validateExposureAngleFullDemo(exposureAngleFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "exposure-angle sidecar report validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "exposure_angle_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "exposure_angle_full_demo_rejected",
			})
			return 14
		}
	}
	if opts.aiCoachCoverValidation {
		if deps.coverFullDemo == nil || deps.validateCoverFullDemo == nil {
			fmt.Fprintln(stderr, "cover sidecar: builder or validator is unavailable")
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "cover_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "cover_full_demo_builder_unavailable",
			})
			return 15
		}
		coverRowsPath := filepath.Join(exportRoot, parserpkg.ObservedCoverRowsFilename)
		coverBytes, coverSHA256, buildErr := writeDeterministicGzipArtifact(
			coverRowsPath,
			func(writer io.Writer) error {
				var err error
				coverFullDemoReport, err = deps.coverFullDemo(
					parsed,
					opts.sourceAlias,
					opts.checksum,
					opts.mapsRoot,
					opts.coverPackagesRoot,
					writer,
				)
				return err
			},
		)
		if buildErr != nil {
			fmt.Fprintf(stderr, "cover sidecar validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "cover_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "cover_full_demo_rejected",
			})
			return 15
		}
		coverFullDemoReport.BuildIdentifier = opts.buildIDValue
		coverFullDemoReport.RowsArtifact.Path = parserpkg.ObservedCoverRowsFilename
		coverFullDemoReport.RowsArtifact.Bytes = coverBytes
		coverFullDemoReport.RowsArtifact.SHA256 = coverSHA256
		if err := deps.validateCoverFullDemo(coverFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "cover sidecar report validation failed: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "cover_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "cover_full_demo_rejected",
			})
			return 15
		}
	}
	if opts.aiCoachExitRouteValidation {
		if deps.exitRouteFullDemo == nil || deps.validateExitRouteFullDemo == nil {
			fmt.Fprintln(stderr, "exit-route sidecar: builder or validator is unavailable")
			emitResult(stdout, commandResult{Status: "failed", Stage: "exit_route_full_demo", SourceAlias: opts.sourceAlias, ErrorCode: "exit_route_full_demo_builder_unavailable"})
			return 16
		}
		exitRowsPath := filepath.Join(exportRoot, parserpkg.ObservedExitRouteRowsFilename)
		exitBytes, exitSHA256, buildErr := writeDeterministicGzipArtifact(exitRowsPath, func(writer io.Writer) error {
			var err error
			exitRouteFullDemoReport, err = deps.exitRouteFullDemo(parsed, opts.sourceAlias, opts.checksum, opts.mapsRoot, opts.exitRoutePackagesRoot, writer)
			return err
		})
		if buildErr != nil {
			fmt.Fprintf(stderr, "exit-route sidecar validation failed: %v\n", buildErr)
			emitResult(stdout, commandResult{Status: "failed", Stage: "exit_route_full_demo", SourceAlias: opts.sourceAlias, ErrorCode: "exit_route_full_demo_rejected"})
			return 16
		}
		exitRouteFullDemoReport.BuildIdentifier = opts.buildIDValue
		exitRouteFullDemoReport.RowsArtifact.Path = parserpkg.ObservedExitRouteRowsFilename
		exitRouteFullDemoReport.RowsArtifact.Bytes = exitBytes
		exitRouteFullDemoReport.RowsArtifact.SHA256 = exitSHA256
		if err := deps.validateExitRouteFullDemo(exitRouteFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "exit-route sidecar report validation failed: %v\n", err)
			emitResult(stdout, commandResult{Status: "failed", Stage: "exit_route_full_demo", SourceAlias: opts.sourceAlias, ErrorCode: "exit_route_full_demo_rejected"})
			return 16
		}
	}

	provenance := models.CanonicalExportProvenance{
		Source:          "demo",
		DemoChecksum:    opts.checksum,
		BuildIdentifier: opts.buildIDValue,
	}
	if err := deps.export(parsed.Context, opts.sourceAlias, exportRoot, "", provenance); err != nil {
		fmt.Fprintf(stderr, "export canonical bundle: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "export", SourceAlias: opts.sourceAlias,
			ErrorCode: "canonical_export_rejected",
		})
		return 5
	}

	bundleName := "match_" + opts.sourceAlias
	stagedPublication := filepath.Join(stagingRoot, publicationName)
	finalPublication := filepath.Join(opts.outputRoot, publicationName)
	smokeReportName := ""
	if navigationSmokeEnabled {
		navigationSmokeReport.BuildIdentifier = opts.buildIDValue
		navigationSmokeReport.ProcessingDurationMS = time.Since(startedAt).Milliseconds()
		payload, err := json.MarshalIndent(navigationSmokeReport, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode navigation smoke report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_smoke_report_failed",
			})
			return 7
		}
		smokeReportName = "observed_navigation_smoke.json"
		if err := os.WriteFile(filepath.Join(exportRoot, smokeReportName), append(payload, '\n'), 0o600); err != nil {
			fmt.Fprintf(stderr, "write navigation smoke report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_smoke_report_failed",
			})
			return 7
		}
	}
	visualContactSmokeReportName := ""
	if opts.aiCoachSmoke || opts.aiCoachFullDemoValidation {
		visualContactSmokeReport.BuildIdentifier = opts.buildIDValue
		visualContactSmokeReport.ProcessingDurationMS = time.Since(startedAt).Milliseconds()
		payload, err := json.MarshalIndent(visualContactSmokeReport, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode visual contact smoke report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_smoke_report_failed",
			})
			return 8
		}
		visualContactSmokeReportName = "observed_visual_contact_smoke.json"
		if err := os.WriteFile(filepath.Join(exportRoot, visualContactSmokeReportName), append(payload, '\n'), 0o600); err != nil {
			fmt.Fprintf(stderr, "write visual contact smoke report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_smoke", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_smoke_report_failed",
			})
			return 8
		}
	}
	navigationFullDemoReportName := ""
	visualContactFullDemoReportName := ""
	weaponStateFullDemoReportName := ""
	utilityInitialVelocityFullDemoReportName := ""
	lastKnownPositionFullDemoReportName := ""
	exposureAngleFullDemoReportName := ""
	coverFullDemoReportName := ""
	exitRouteFullDemoReportName := ""
	if opts.aiCoachFullDemoValidation {
		navigationFullDemoReportName = parserpkg.ObservedNavigationFullDemoReportFilename
		if err := writeCompactJSONReport(
			filepath.Join(exportRoot, navigationFullDemoReportName),
			navigationFullDemoReport,
		); err != nil {
			fmt.Fprintf(stderr, "write full-demo navigation report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "navigation_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "navigation_full_demo_report_failed",
			})
			return 9
		}

		visualContactFullDemoReportName = parserpkg.ObservedVisualContactFullDemoReportFilename
		if err := writeCompactJSONReport(
			filepath.Join(exportRoot, visualContactFullDemoReportName),
			visualContactFullDemoReport,
		); err != nil {
			fmt.Fprintf(stderr, "write full-demo visual-contact report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "visual_contact_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "visual_contact_full_demo_report_failed",
			})
			return 10
		}

		weaponStateFullDemoReportName = parserpkg.ObservedWeaponStateFullDemoReportFilename
		if err := writeCompactJSONReport(
			filepath.Join(exportRoot, weaponStateFullDemoReportName),
			weaponStateFullDemoReport,
		); err != nil {
			fmt.Fprintf(stderr, "write full-demo weapon-state report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "weapon_state_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "weapon_state_full_demo_report_failed",
			})
			return 11
		}

		utilityInitialVelocityFullDemoReportName = parserpkg.ObservedUtilityInitialVelocityFullDemoReportFilename
		if err := writeCompactJSONReport(
			filepath.Join(exportRoot, utilityInitialVelocityFullDemoReportName),
			utilityInitialVelocityFullDemoReport,
		); err != nil {
			fmt.Fprintf(stderr, "write full-demo utility initial-velocity report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "utility_initial_velocity_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "utility_initial_velocity_full_demo_report_failed",
			})
			return 12
		}

		lastKnownPositionFullDemoReportName = parserpkg.ObservedLastKnownPositionFullDemoReportFilename
		if err := writeCompactJSONReport(
			filepath.Join(exportRoot, lastKnownPositionFullDemoReportName),
			lastKnownPositionFullDemoReport,
		); err != nil {
			fmt.Fprintf(stderr, "write full-demo last-known position report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "last_known_position_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "last_known_position_full_demo_report_failed",
			})
			return 13
		}
	}
	if opts.aiCoachExposureValidation {
		exposureAngleFullDemoReportName = parserpkg.ObservedExposureAngleFullDemoFilename
		if err := writeCompactJSONReport(
			filepath.Join(exportRoot, exposureAngleFullDemoReportName),
			exposureAngleFullDemoReport,
		); err != nil {
			fmt.Fprintf(stderr, "write exposure-angle full-demo report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "exposure_angle_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "exposure_angle_full_demo_report_failed",
			})
			return 14
		}
	}
	if opts.aiCoachCoverValidation {
		coverFullDemoReportName = parserpkg.ObservedCoverFullDemoFilename
		if err := writeCompactJSONReport(
			filepath.Join(exportRoot, coverFullDemoReportName),
			coverFullDemoReport,
		); err != nil {
			fmt.Fprintf(stderr, "write cover full-demo report: %v\n", err)
			emitResult(stdout, commandResult{
				Status: "failed", Stage: "cover_full_demo", SourceAlias: opts.sourceAlias,
				ErrorCode: "cover_full_demo_report_failed",
			})
			return 15
		}
	}
	if opts.aiCoachExitRouteValidation {
		exitRouteFullDemoReportName = parserpkg.ObservedExitRouteFullDemoFilename
		if err := writeCompactJSONReport(filepath.Join(exportRoot, exitRouteFullDemoReportName), exitRouteFullDemoReport); err != nil {
			fmt.Fprintf(stderr, "write exit-route full-demo report: %v\n", err)
			emitResult(stdout, commandResult{Status: "failed", Stage: "exit_route_full_demo", SourceAlias: opts.sourceAlias, ErrorCode: "exit_route_full_demo_report_failed"})
			return 16
		}
	}
	if !smokeEnabled {
		// Legacy canonical-only publication stays unchanged.
		stagedPublication = filepath.Join(stagingRoot, bundleName)
		finalPublication = filepath.Join(opts.outputRoot, bundleName)
	}
	if opts.aiCoachSpatialCandidate {
		bridge, err := parserpkg.BuildSpatialParticipantBridge(parsed.ReplayData, opts.sourceAlias, opts.checksum)
		if err == nil {
			err = writeCompactJSONReport(filepath.Join(exportRoot, "spatial_participant_bridge.json"), bridge)
		}
		if err != nil {
			fmt.Fprintf(stderr, "spatial participant bridge: %v\n", err)
			return 17
		}
	}
	if opts.aiCoachEconomyCandidate {
		if err := writeEconomyCandidate(parsed.Context, opts, exportRoot, bundleName); err != nil {
			fmt.Fprintf(stderr, "economic candidate rejected: %v\n", err)
			emitResult(stdout, commandResult{Status: "failed", Stage: "economy_candidate", SourceAlias: opts.sourceAlias, ErrorCode: "economy_candidate_rejected"})
			return 18
		}
	}
	if opts.aiCoachSupportCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportObservations(parsed.Context, opts, exportRoot, bundleName); err != nil {
			fmt.Fprintf(stderr, "support candidate rejected: %v\n", err)
			return 19
		}
	}
	if opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportPhysical(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support physical candidate rejected: %v\n", err)
			return 20
		}
	}
	if opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportObjective(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support objective candidate rejected: %v\n", err)
			return 21
		}
	}
	if opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportC4(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support C4 candidate rejected: %v\n", err)
			return 22
		}
	}
	if opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportObjectiveClock(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support objective clock export failed: %v\n", err)
			return 5
		}
	}
	if opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportAttemptOrigin(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support attempt origin export failed: %v\n", err)
			return 5
		}
	}
	if opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportObjectivePosition(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support objective position export failed: %v\n", err)
			return 5
		}
	}
	if opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportNativeMotion(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support native motion export failed: %v\n", err)
			return 5
		}
	}
	if opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportWeaponTiming(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support weapon timing export failed: %v\n", err)
			return 5
		}
	}
	if opts.aiCoachSupportVisualContactCandidate {
		if err := writeSupportVisualContact(parsed.Context, opts, exportRoot); err != nil {
			fmt.Fprintf(stderr, "support visual contact export failed: %v\n", err)
			return 5
		}
	}
	if err := os.Rename(stagedPublication, finalPublication); err != nil {
		fmt.Fprintf(stderr, "publish canonical bundle: %v\n", err)
		emitResult(stdout, commandResult{
			Status: "failed", Stage: "publication", SourceAlias: opts.sourceAlias,
			ErrorCode: "atomic_publish_failed",
		})
		return 6
	}

	bundleDir := bundleName
	reportPath := ""
	visualContactReportPath := ""
	navigationFullDemoReportPath := ""
	navigationFullDemoRowsPath := ""
	visualContactFullDemoReportPath := ""
	visualContactFullDemoAuditPath := ""
	weaponStateFullDemoReportPath := ""
	weaponStateFullDemoRowsPath := ""
	utilityInitialVelocityFullDemoReportPath := ""
	utilityInitialVelocityFullDemoRowsPath := ""
	lastKnownPositionFullDemoReportPath := ""
	lastKnownPositionFullDemoRowsPath := ""
	exposureAngleFullDemoReportPath := ""
	exposureAngleFullDemoRowsPath := ""
	coverFullDemoReportPath := ""
	coverFullDemoRowsPath := ""
	exitRouteFullDemoReportPath := ""
	exitRouteFullDemoRowsPath := ""
	economyCandidateManifestPath := ""
	if opts.aiCoachEconomyCandidate {
		economyCandidateManifestPath = filepath.Join(publicationName, parserpkg.EconomyCandidateManifestFilename)
	}
	if smokeEnabled {
		bundleDir = filepath.Join(publicationName, bundleName)
		if smokeReportName != "" {
			reportPath = filepath.Join(publicationName, smokeReportName)
		}
		if visualContactSmokeReportName != "" {
			visualContactReportPath = filepath.Join(publicationName, visualContactSmokeReportName)
		}
		if opts.aiCoachExposureValidation {
			exposureAngleFullDemoReportPath = filepath.Join(publicationName, exposureAngleFullDemoReportName)
			exposureAngleFullDemoRowsPath = filepath.Join(publicationName, parserpkg.ObservedExposureAngleRowsFilename)
		}
		if opts.aiCoachCoverValidation {
			coverFullDemoReportPath = filepath.Join(publicationName, coverFullDemoReportName)
			coverFullDemoRowsPath = filepath.Join(publicationName, parserpkg.ObservedCoverRowsFilename)
		}
		if opts.aiCoachExitRouteValidation {
			exitRouteFullDemoReportPath = filepath.Join(publicationName, exitRouteFullDemoReportName)
			exitRouteFullDemoRowsPath = filepath.Join(publicationName, parserpkg.ObservedExitRouteRowsFilename)
		}
		if opts.aiCoachFullDemoValidation {
			navigationFullDemoReportPath = filepath.Join(publicationName, navigationFullDemoReportName)
			navigationFullDemoRowsPath = filepath.Join(publicationName, parserpkg.ObservedNavigationFullDemoRowsFilename)
			visualContactFullDemoReportPath = filepath.Join(publicationName, visualContactFullDemoReportName)
			visualContactFullDemoAuditPath = filepath.Join(publicationName, parserpkg.ObservedVisualContactFullDemoAuditFilename)
			weaponStateFullDemoReportPath = filepath.Join(publicationName, weaponStateFullDemoReportName)
			weaponStateFullDemoRowsPath = filepath.Join(publicationName, parserpkg.ObservedWeaponStateFullDemoRowsFilename)
			utilityInitialVelocityFullDemoReportPath = filepath.Join(publicationName, utilityInitialVelocityFullDemoReportName)
			utilityInitialVelocityFullDemoRowsPath = filepath.Join(publicationName, parserpkg.ObservedUtilityInitialVelocityFullDemoRowsFilename)
			lastKnownPositionFullDemoReportPath = filepath.Join(publicationName, lastKnownPositionFullDemoReportName)
			lastKnownPositionFullDemoRowsPath = filepath.Join(publicationName, parserpkg.ObservedLastKnownPositionFullDemoRowsFilename)
		}
	}
	emitResult(stdout, commandResult{
		SchemaID: "stratai.golden_demo_export_result@1", Status: "exported", Stage: "complete",
		EconomyCandidateManifest: economyCandidateManifestPath,
		SourceAlias:              opts.sourceAlias, SHA256: opts.checksum, BundleDir: bundleDir,
		SmokeReport: reportPath, VisualContactSmokeReport: visualContactReportPath,
		NavigationFullDemoReport:             navigationFullDemoReportPath,
		NavigationFullDemoRows:               navigationFullDemoRowsPath,
		VisualContactFullDemoReport:          visualContactFullDemoReportPath,
		VisualContactFullDemoAudit:           visualContactFullDemoAuditPath,
		WeaponStateFullDemoReport:            weaponStateFullDemoReportPath,
		WeaponStateFullDemoRows:              weaponStateFullDemoRowsPath,
		UtilityInitialVelocityFullDemoReport: utilityInitialVelocityFullDemoReportPath,
		UtilityInitialVelocityFullDemoRows:   utilityInitialVelocityFullDemoRowsPath,
		LastKnownPositionFullDemoReport:      lastKnownPositionFullDemoReportPath,
		LastKnownPositionFullDemoRows:        lastKnownPositionFullDemoRowsPath,
		ExposureAngleFullDemoReport:          exposureAngleFullDemoReportPath,
		ExposureAngleFullDemoRows:            exposureAngleFullDemoRowsPath,
		CoverFullDemoReport:                  coverFullDemoReportPath,
		CoverFullDemoRows:                    coverFullDemoRowsPath,
		ExitRouteFullDemoReport:              exitRouteFullDemoReportPath,
		ExitRouteFullDemoRows:                exitRouteFullDemoRowsPath,
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
	flags.StringVar(&opts.exposurePackagesRoot, "exposure-packages-root", "", "versioned exposure package registry root")
	flags.StringVar(&opts.coverPackagesRoot, "cover-packages-root", "", "versioned cover package registry root")
	flags.StringVar(&opts.exitRoutePackagesRoot, "exit-route-packages-root", "", "versioned exit-route package registry root")
	flags.StringVar(&opts.buildIDValue, "build-id", buildID, "deterministic build identifier")
	flags.BoolVar(&opts.navigationSmoke, "navigation-smoke", false, "validate candidate navigation timing on the parsed demo")
	flags.BoolVar(&opts.aiCoachSmoke, "ai-coach-smoke", false, "validate all implemented AI Coach candidate facts on the parsed demo")
	flags.BoolVar(&opts.aiCoachFullDemoValidation, "ai-coach-full-demo-validation", false, "validate implemented AI Coach candidate facts across the complete parsed demo")
	flags.BoolVar(&opts.aiCoachExposureValidation, "ai-coach-exposure-angle-validation", false, "validate the candidate exposure-angle sidecar across the parsed demo")
	flags.BoolVar(&opts.aiCoachCoverValidation, "ai-coach-cover-validation", false, "validate the candidate objective cover sidecar across the parsed demo")
	flags.BoolVar(&opts.aiCoachExitRouteValidation, "ai-coach-exit-route-validation", false, "validate the candidate objective exit-route sidecar across the parsed demo")
	flags.BoolVar(&opts.aiCoachSpatialCandidate, "ai-coach-spatial-candidate", false, "emit all three spatial fact sidecars and opaque join bridge in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachEconomyCandidate, "ai-coach-economy-candidate", false, "capture isolated economic observations and reconcile them in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportCandidate, "ai-coach-support-candidate", false, "capture isolated causal support snapshots in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportPhysicalCandidate, "ai-coach-support-physical-candidate", false, "capture support v1 and exact physical context in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportObjectiveCandidate, "ai-coach-support-objective-candidate", false, "capture support, physical and causal objective context in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportAttemptOriginCandidate, "ai-coach-support-attempt-origin-candidate", false, "capture native arming origins and all previous support layers in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportObjectivePositionCandidate, "ai-coach-support-objective-position-candidate", false, "capture native objective geometry and legal availability evidence and all previous support layers in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportNativeMotionCandidate, "ai-coach-support-native-motion-candidate", false, "capture native pawn motion and ground-reference evidence and all previous support layers in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportVisualContactCandidate, "ai-coach-support-visual-contact-candidate", false, "capture exact support visual inputs and isolated static ray diagnostics with explicit legal-contact abstentions")
	flags.BoolVar(&opts.aiCoachSupportWeaponTimingCandidate, "ai-coach-support-weapon-timing-candidate", false, "capture native weapon timing and current ownership evidence and all previous support layers in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportObjectiveClockCandidate, "ai-coach-support-objective-clock-candidate", false, "capture native objective clocks and previous support layers in one temporary demo pass")
	flags.BoolVar(&opts.aiCoachSupportC4Candidate, "ai-coach-support-c4-candidate", false, "capture support, physical, objective and native C4 ownership in one temporary demo pass")
	flags.IntVar(&opts.economyCandidateVersion, "economy-candidate-version", 1, "isolated economy mode (1: observations, 2: initial frontiers, 3: native purchase sources, 4: native rule context, 5: session payloads, 6: resource manifest, 7: transaction trace, 8: native identity and clocks, 9: frame-end ownership, 10: entries at ownership cuts, 11: cashflow at common cuts)")
	flags.IntVar(&opts.navigationSmokeQueries, "navigation-smoke-queries", 64, "bounded real navigation queries (1-256)")
	flags.IntVar(&opts.visualContactSmokeFrames, "visual-contact-smoke-frames", 256, "bounded causal real visual-contact round-prefix frames (16-256)")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("positional arguments are not allowed")
	}
	if opts.economyCandidateVersion < 1 || opts.economyCandidateVersion > 11 || (opts.economyCandidateVersion >= 2 && !opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("economic candidate version requires standalone economy mode and version 1 through 8")
	}
	if strings.TrimSpace(opts.demoPath) == "" || strings.TrimSpace(opts.outputRoot) == "" ||
		strings.TrimSpace(opts.sourceAlias) == "" || strings.TrimSpace(opts.checksum) == "" {
		return options{}, errors.New("--demo, --output, --alias and --checksum are required")
	}
	if strings.TrimSpace(opts.buildIDValue) == "" {
		return options{}, errors.New("--build-id cannot be empty")
	}
	if opts.navigationSmokeQueries < 1 || opts.navigationSmokeQueries > 256 {
		return options{}, errors.New("--navigation-smoke-queries must be between 1 and 256")
	}
	if opts.visualContactSmokeFrames < 16 || opts.visualContactSmokeFrames > 256 {
		return options{}, errors.New("--visual-contact-smoke-frames must be between 16 and 256")
	}
	if opts.navigationSmoke && opts.aiCoachSmoke {
		return options{}, errors.New("--ai-coach-smoke already includes --navigation-smoke")
	}
	if opts.aiCoachFullDemoValidation && (opts.navigationSmoke || opts.aiCoachSmoke) {
		return options{}, errors.New("--ai-coach-full-demo-validation is a standalone validation mode")
	}
	if opts.aiCoachExposureValidation && (opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation) {
		return options{}, errors.New("--ai-coach-exposure-angle-validation is a standalone validation mode")
	}
	if opts.aiCoachCoverValidation && (opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachExposureValidation) {
		return options{}, errors.New("--ai-coach-cover-validation is a standalone validation mode")
	}
	if opts.aiCoachExitRouteValidation && (opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation) {
		return options{}, errors.New("--ai-coach-exit-route-validation is a standalone validation mode")
	}
	if opts.aiCoachExposureValidation && opts.aiCoachCoverValidation {
		return options{}, errors.New("exposure-angle and cover validation modes are mutually exclusive")
	}
	if opts.aiCoachSpatialCandidate {
		if opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation {
			return options{}, errors.New("--ai-coach-spatial-candidate is a standalone candidate mode")
		}
		opts.aiCoachExposureValidation, opts.aiCoachCoverValidation, opts.aiCoachExitRouteValidation = true, true, true
	}
	if opts.aiCoachEconomyCandidate && (opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation) {
		return options{}, errors.New("--ai-coach-economy-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportCandidate && (opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportPhysicalCandidate && (opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-physical-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportObjectiveCandidate && (opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-objective-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportC4Candidate && (opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-c4-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportVisualContactCandidate && (opts.aiCoachSupportWeaponTimingCandidate || opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-visual-contact-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportWeaponTimingCandidate && (opts.aiCoachSupportNativeMotionCandidate || opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-weapon-timing-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportNativeMotionCandidate && (opts.aiCoachSupportObjectivePositionCandidate || opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-native-motion-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportObjectivePositionCandidate && (opts.aiCoachSupportAttemptOriginCandidate || opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-objective-position-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportAttemptOriginCandidate && (opts.aiCoachSupportObjectiveClockCandidate || opts.aiCoachSupportC4Candidate || opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate) {
		return options{}, errors.New("--ai-coach-support-attempt-origin-candidate is a standalone candidate mode")
	}
	if opts.aiCoachSupportObjectiveClockCandidate && (opts.aiCoachSupportC4Candidate || (opts.aiCoachSupportObjectiveCandidate || opts.aiCoachSupportPhysicalCandidate || opts.aiCoachSupportCandidate || opts.navigationSmoke || opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachSpatialCandidate || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation || opts.aiCoachEconomyCandidate)) {
		return options{}, errors.New("--ai-coach-support-objective-clock-candidate is a standalone candidate mode")
	}
	if (opts.aiCoachSmoke || opts.aiCoachFullDemoValidation || opts.aiCoachExposureValidation || opts.aiCoachCoverValidation || opts.aiCoachExitRouteValidation) && strings.TrimSpace(opts.mapsRoot) == "" {
		return options{}, errors.New("--maps-root is required for AI Coach geometry validation")
	}
	if opts.aiCoachExposureValidation && strings.TrimSpace(opts.exposurePackagesRoot) == "" {
		return options{}, errors.New("--exposure-packages-root is required for exposure-angle validation")
	}
	if opts.aiCoachCoverValidation && strings.TrimSpace(opts.coverPackagesRoot) == "" {
		return options{}, errors.New("--cover-packages-root is required for cover validation")
	}
	if opts.aiCoachExitRouteValidation && strings.TrimSpace(opts.exitRoutePackagesRoot) == "" {
		return options{}, errors.New("--exit-route-packages-root is required for exit-route validation")
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

func prepareOutputRoot(root, publicationName string, requireEmpty bool) error {
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
	if requireEmpty {
		entries, err := os.ReadDir(root)
		if err != nil {
			return fmt.Errorf("inspect isolated output root: %w", err)
		}
		if len(entries) != 0 {
			return errors.New("navigation smoke output root must be empty")
		}
	}
	finalPublication := filepath.Join(root, publicationName)
	if _, err := os.Lstat(finalPublication); err == nil {
		return errors.New("output publication already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func validateNavigationSmokeOutputRoot(root string) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("navigation smoke output root is required")
	}
	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return fmt.Errorf("resolve system temporary directory: %w", err)
	}
	tempRoot, err = filepath.EvalSymlinks(tempRoot)
	if err != nil {
		return fmt.Errorf("physically resolve system temporary directory: %w", err)
	}
	tempRoot, err = filepath.Abs(tempRoot)
	if err != nil {
		return fmt.Errorf("normalize physical system temporary directory: %w", err)
	}
	outputRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve output root: %w", err)
	}
	logicalTempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return fmt.Errorf("normalize system temporary directory: %w", err)
	}
	logicalRelative, err := strictPathDescendant(logicalTempRoot, outputRoot)
	if err != nil {
		return fmt.Errorf("compare output with system temporary directory: %w", err)
	}
	if logicalRelative == "" {
		return errors.New("navigation smoke output must be a dedicated directory below the system temporary directory")
	}

	current := logicalTempRoot
	for _, component := range strings.Split(logicalRelative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return fmt.Errorf("inspect smoke output path component: %w", statErr)
		}
		reparse, reparseErr := pathIsReparsePoint(current, info)
		if reparseErr != nil {
			return fmt.Errorf("inspect smoke output path component attributes: %w", reparseErr)
		}
		if reparse || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("navigation smoke output path must not contain symlinks or reparse points")
		}
		if !info.IsDir() {
			return errors.New("navigation smoke output path components must be directories")
		}
	}

	physicalOutputRoot, err := filepath.EvalSymlinks(outputRoot)
	if err != nil {
		return fmt.Errorf("physically resolve output root: %w", err)
	}
	physicalOutputRoot, err = filepath.Abs(physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("normalize physical output root: %w", err)
	}
	physicalRelative, err := strictPathDescendant(tempRoot, physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("compare physical output with physical system temporary directory: %w", err)
	}
	if physicalRelative == "" {
		return errors.New("navigation smoke output must physically remain below the system temporary directory")
	}
	expectedPhysicalRoot := filepath.Join(tempRoot, logicalRelative)
	expectedInfo, err := os.Stat(expectedPhysicalRoot)
	if err != nil {
		return fmt.Errorf("inspect expected physical output root: %w", err)
	}
	physicalInfo, err := os.Stat(physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("inspect physical output root: %w", err)
	}
	if !os.SameFile(expectedInfo, physicalInfo) {
		return errors.New("navigation smoke output physical resolution differs from its temporary path")
	}
	entries, err := os.ReadDir(physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("inspect isolated physical output root: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("navigation smoke output root must be empty")
	}
	return nil
}

func strictPathDescendant(root, target string) (string, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if relative == "." || relative == ".." || filepath.IsAbs(relative) ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", nil
	}
	return relative, nil
}

type navigationGraphProvider interface {
	NavigationGraphSnapshot() (*navigation.Graph, error)
}

func buildNavigationSmokeReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	queryLimit int,
) (parserpkg.ObservedNavigationSmokeReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedNavigationSmokeReport{}, errors.New("parsed context or replay data is unavailable")
	}
	provider, ok := parsed.Context.MapManager.(navigationGraphProvider)
	if !ok || provider == nil {
		return parserpkg.ObservedNavigationSmokeReport{}, errors.New("map manager does not expose a NAV graph")
	}
	graph, err := provider.NavigationGraphSnapshot()
	if err != nil {
		return parserpkg.ObservedNavigationSmokeReport{}, fmt.Errorf("snapshot NAV graph: %w", err)
	}
	return parserpkg.BuildObservedNavigationSmokeReport(
		parsed.ReplayData,
		graph,
		sourceAlias,
		sourceSHA256,
		queryLimit,
	)
}

func buildNavigationFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	rowsWriter io.Writer,
) (parserpkg.ObservedNavigationFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedNavigationFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	provider, ok := parsed.Context.MapManager.(navigationGraphProvider)
	if !ok || provider == nil {
		return parserpkg.ObservedNavigationFullDemoReport{}, errors.New("map manager does not expose a NAV graph")
	}
	graph, err := provider.NavigationGraphSnapshot()
	if err != nil {
		return parserpkg.ObservedNavigationFullDemoReport{}, fmt.Errorf("snapshot NAV graph: %w", err)
	}
	return parserpkg.BuildObservedNavigationFullDemoReport(
		parsed.ReplayData,
		graph,
		sourceAlias,
		sourceSHA256,
		rowsWriter,
	)
}

func buildObservedWeaponStateFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	rowsWriter io.Writer,
) (parserpkg.ObservedWeaponStateFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedWeaponStateFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	if parsed.Context.Combat == nil {
		return parserpkg.ObservedWeaponStateFullDemoReport{}, errors.New("combat ledger is unavailable")
	}
	return parserpkg.BuildObservedWeaponStateFullDemoReport(
		parsed.ReplayData,
		parsed.Context.Combat.Snapshot(),
		sourceAlias,
		sourceSHA256,
		rowsWriter,
	)
}

func buildObservedUtilityInitialVelocityFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	rowsWriter io.Writer,
) (parserpkg.ObservedUtilityInitialVelocityFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedUtilityInitialVelocityFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	if parsed.Context.Utilities == nil {
		return parserpkg.ObservedUtilityInitialVelocityFullDemoReport{}, errors.New("utility ledger is unavailable")
	}
	return parserpkg.BuildObservedUtilityInitialVelocityFullDemoReport(
		parsed.Context.Utilities.Snapshot(),
		sourceAlias,
		sourceSHA256,
		rowsWriter,
	)
}

func buildObservedLastKnownPositionFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	rowsWriter io.Writer,
) (parserpkg.ObservedLastKnownPositionFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedLastKnownPositionFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	geometry, ok := parsed.Context.MapManager.(parserpkg.TacticalGeometry)
	if !ok || geometry == nil || !geometry.IsLoaded() {
		return parserpkg.ObservedLastKnownPositionFullDemoReport{}, errors.New("loaded tactical geometry is unavailable")
	}
	return parserpkg.BuildObservedLastKnownPositionFullDemoReport(
		parsed.ReplayData,
		geometry,
		sourceAlias,
		sourceSHA256,
		rowsWriter,
	)
}

type mapInputProvenanceProvider interface {
	InputProvenance() map[string]mapspkg.InputProvenance
}

func buildObservedExposureAngleFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	mapsRoot string,
	exposurePackagesRoot string,
	rowsWriter io.Writer,
) (parserpkg.ObservedExposureAngleFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedExposureAngleFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	mapID := strings.TrimSpace(parsed.ReplayData.Metadata.MapName)
	resolution := exposure.ResolvePackage(
		exposurePackagesRoot,
		mapsRoot,
		mapID,
		exposure.ContractVersion,
		"",
	)
	return parserpkg.BuildObservedExposureAngleFullDemoReport(
		parsed.ReplayData,
		resolution,
		sourceAlias,
		sourceSHA256,
		rowsWriter,
	)
}

func buildObservedCoverFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	mapsRoot string,
	coverPackagesRoot string,
	rowsWriter io.Writer,
) (parserpkg.ObservedCoverFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedCoverFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	mapID := strings.TrimSpace(parsed.ReplayData.Metadata.MapName)
	resolution := cover.ResolvePackage(
		coverPackagesRoot,
		mapsRoot,
		mapID,
		cover.ContractVersion,
		"",
	)
	return parserpkg.BuildObservedCoverFullDemoReport(
		parsed.ReplayData,
		resolution,
		sourceAlias,
		sourceSHA256,
		rowsWriter,
	)
}

func buildObservedExitRouteFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	mapsRoot string,
	exitRoutePackagesRoot string,
	rowsWriter io.Writer,
) (parserpkg.ObservedExitRouteFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedExitRouteFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	provider, ok := parsed.Context.MapManager.(interface {
		NavigationGraphSnapshot() (*navigation.Graph, error)
	})
	if !ok {
		return parserpkg.ObservedExitRouteFullDemoReport{}, errors.New("loaded directed NAV graph provider is unavailable")
	}
	graph, err := provider.NavigationGraphSnapshot()
	if err != nil || graph == nil {
		return parserpkg.ObservedExitRouteFullDemoReport{}, errors.New("loaded directed NAV graph is unavailable")
	}
	mapID := strings.TrimSpace(parsed.ReplayData.Metadata.MapName)
	resolution := exitroute.ResolvePackage(exitRoutePackagesRoot, mapsRoot, mapID, exitroute.ContractVersion, "")
	return parserpkg.BuildObservedExitRouteFullDemoReport(parsed.ReplayData, graph, resolution, sourceAlias, sourceSHA256, rowsWriter)
}

func buildVisualContactSmokeReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	mapsRoot string,
	frameLimit int,
) (parserpkg.ObservedVisualContactSmokeReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedVisualContactSmokeReport{}, errors.New("parsed context or replay data is unavailable")
	}
	geometry, ok := parsed.Context.MapManager.(parserpkg.TacticalGeometry)
	if !ok || geometry == nil || !geometry.IsLoaded() {
		return parserpkg.ObservedVisualContactSmokeReport{}, errors.New("loaded tactical geometry is unavailable")
	}
	assets, err := visualContactGeometryAssets(parsed.Context.MapManager, mapsRoot)
	if err != nil {
		return parserpkg.ObservedVisualContactSmokeReport{}, err
	}
	return parserpkg.BuildObservedVisualContactSmokeReport(
		parsed.ReplayData,
		geometry,
		sourceAlias,
		sourceSHA256,
		frameLimit,
		assets,
	)
}

func buildVisualContactFullDemoReport(
	parsed *parserpkg.ParseDemoResult,
	sourceAlias string,
	sourceSHA256 string,
	mapsRoot string,
	auditWriter io.Writer,
) (parserpkg.ObservedVisualContactFullDemoReport, error) {
	if parsed == nil || parsed.Context == nil || parsed.ReplayData == nil {
		return parserpkg.ObservedVisualContactFullDemoReport{}, errors.New("parsed context or replay data is unavailable")
	}
	geometry, ok := parsed.Context.MapManager.(parserpkg.TacticalGeometry)
	if !ok || geometry == nil || !geometry.IsLoaded() {
		return parserpkg.ObservedVisualContactFullDemoReport{}, errors.New("loaded tactical geometry is unavailable")
	}
	assets, err := visualContactGeometryAssets(parsed.Context.MapManager, mapsRoot)
	if err != nil {
		return parserpkg.ObservedVisualContactFullDemoReport{}, err
	}
	return parserpkg.BuildObservedVisualContactFullDemoReport(
		parsed.ReplayData,
		geometry,
		sourceAlias,
		sourceSHA256,
		assets,
		auditWriter,
	)
}

func visualContactGeometryAssets(
	manager mapspkg.VisibilityChecker,
	mapsRoot string,
) ([]parserpkg.ObservedVisualContactSmokeAsset, error) {
	provider, ok := manager.(mapInputProvenanceProvider)
	if !ok || provider == nil {
		return nil, errors.New("map manager does not expose physics provenance")
	}
	physics, ok := provider.InputProvenance()["physics_map"]
	if !ok || !physics.Used || physics.LoadError != "" || strings.TrimSpace(physics.RelativePath) == "" ||
		!checksumPattern.MatchString(physics.SHA256) {
		return nil, errors.New("loaded physics GLTF provenance is incomplete")
	}

	root, err := filepath.Abs(mapsRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve maps root: %w", err)
	}
	gltfPath, gltfReference, err := resolveSmokeMapAsset(root, physics.RelativePath)
	if err != nil {
		return nil, fmt.Errorf("resolve physics GLTF: %w", err)
	}
	gltfSHA256, err := sha256File(gltfPath)
	if err != nil {
		return nil, fmt.Errorf("hash physics GLTF: %w", err)
	}
	if gltfSHA256 != physics.SHA256 {
		return nil, errors.New("physics GLTF changed after map loading")
	}

	payload, err := os.ReadFile(gltfPath)
	if err != nil {
		return nil, fmt.Errorf("read physics GLTF manifest: %w", err)
	}
	var document struct {
		Buffers []struct {
			URI string `json:"uri"`
		} `json:"buffers"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("decode physics GLTF manifest: %w", err)
	}
	if len(document.Buffers) == 0 {
		return nil, errors.New("physics GLTF declares no external geometry buffer")
	}

	assets := []parserpkg.ObservedVisualContactSmokeAsset{{
		Reference: gltfReference,
		SHA256:    gltfSHA256,
		Role:      "physics_gltf",
	}}
	seen := map[string]struct{}{gltfReference: {}}
	for _, buffer := range document.Buffers {
		uri := strings.TrimSpace(buffer.URI)
		if uri == "" || strings.HasPrefix(strings.ToLower(uri), "data:") ||
			strings.Contains(uri, "://") || filepath.IsAbs(filepath.FromSlash(uri)) {
			return nil, errors.New("physics GLTF buffer must be a local relative file")
		}
		bufferRelative := filepath.Join(filepath.Dir(gltfReference), filepath.FromSlash(uri))
		bufferPath, bufferReference, err := resolveSmokeMapAsset(root, bufferRelative)
		if err != nil {
			return nil, fmt.Errorf("resolve physics buffer: %w", err)
		}
		if _, duplicate := seen[bufferReference]; duplicate {
			continue
		}
		bufferSHA256, err := sha256File(bufferPath)
		if err != nil {
			return nil, fmt.Errorf("hash physics buffer: %w", err)
		}
		assets = append(assets, parserpkg.ObservedVisualContactSmokeAsset{
			Reference: bufferReference,
			SHA256:    bufferSHA256,
			Role:      "physics_buffer",
		})
		seen[bufferReference] = struct{}{}
	}
	if len(assets) < 2 {
		return nil, errors.New("physics GLTF external buffer provenance is missing")
	}
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].Role != assets[j].Role {
			return assets[i].Role < assets[j].Role
		}
		return assets[i].Reference < assets[j].Reference
	})
	return assets, nil
}

func resolveSmokeMapAsset(root, relative string) (string, string, error) {
	reference := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if reference == "." || filepath.IsAbs(reference) {
		return "", "", errors.New("asset reference must be relative")
	}
	path := filepath.Join(root, reference)
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	within, err := filepath.Rel(root, absolute)
	if err != nil || within == ".." || filepath.IsAbs(within) ||
		strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", "", errors.New("asset reference escapes maps root")
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", "", errors.New("asset must be a regular non-symlink file")
	}
	return absolute, filepath.ToSlash(within), nil
}

func writeCompactJSONReport(path string, report any) error {
	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode compact JSON: %w", err)
	}
	payload = append(payload, '\n')
	handle, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, writeErr := handle.Write(payload)
	closeErr := handle.Close()
	if writeErr == nil && written != len(payload) {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		_ = os.Remove(path)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return closeErr
	}
	return nil
}

type deterministicGzipCountingWriter struct {
	destination io.Writer
	bytes       int64
}

func (writer *deterministicGzipCountingWriter) Write(payload []byte) (int, error) {
	written, err := writer.destination.Write(payload)
	writer.bytes += int64(written)
	if err == nil && written != len(payload) {
		err = io.ErrShortWrite
	}
	return written, err
}

func writeDeterministicGzipArtifact(
	path string,
	produce func(io.Writer) error,
) (int64, string, error) {
	if produce == nil {
		return 0, "", errors.New("gzip producer is nil")
	}
	handle, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, "", err
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.Remove(path)
		}
	}()

	hash := sha256.New()
	compressed := &deterministicGzipCountingWriter{
		destination: io.MultiWriter(handle, hash),
	}
	gzipWriter, err := gzip.NewWriterLevel(compressed, gzip.BestCompression)
	if err != nil {
		_ = handle.Close()
		return 0, "", err
	}
	gzipWriter.Header.ModTime = time.Time{}
	gzipWriter.Header.Name = ""
	gzipWriter.Header.Comment = ""
	gzipWriter.Header.Extra = nil
	gzipWriter.Header.OS = 255

	produceErr := produce(gzipWriter)
	gzipErr := gzipWriter.Close()
	fileErr := handle.Close()
	if produceErr != nil {
		return 0, "", produceErr
	}
	if gzipErr != nil {
		return 0, "", gzipErr
	}
	if fileErr != nil {
		return 0, "", fileErr
	}
	completed = true
	return compressed.bytes, hex.EncodeToString(hash.Sum(nil)), nil
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
