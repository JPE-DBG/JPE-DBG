package main

import (
	"fmt"
	"os"
	"test-tool/step_definitions" // Ensure this path is correct based on your go.mod
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/spf13/pflag" // Godog recommends pflag for options
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress", // "pretty" or "progress"
	Paths:  []string{"features"},
	Strict: true, // Fail if there are undefined or pending steps
	// Tags: "", // Add tags to filter scenarios
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts) // Allow godog CLI flags
}

func TestMain(m *testing.M) {
	pflag.Parse()
	opts.Paths = pflag.Args()
	if len(opts.Paths) == 0 {
		opts.Paths = []string{"features"} // Default to features directory
	}

	status := godog.TestSuite{
		Name:                 "elsa-test-suite",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              &opts,
	}.Run()

	// Optional: Call AfterSuiteHook if you have one defined and need to manage it this way
	// This is a common pattern if AfterSuite is not directly supported by the Godog version's hooks.
	// For this PoC, we'll call it directly.
	step_definitions.AfterSuiteHook()

	if status > 0 {
		fmt.Println("Godog tests failed!")
		// os.Exit(status) // Exiting here might hide Go test summary
	} else {
		fmt.Println("Godog tests passed!")
	}
	os.Exit(status) // Ensure the exit code reflects the test suite status
}

// InitializeTestSuite can be used for global suite setup if needed
func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	// Example: if you need to run something once before all scenarios in all suites
	// For this PoC, most setup is per-scenario or managed by hooks.go
}

// InitializeScenario registers step definitions.
func InitializeScenario(ctx *godog.ScenarioContext) {
	// Initialize steps from all your step definition files
	step_definitions.InitializeConfigSteps(ctx)
	step_definitions.InitializeMessageSteps(ctx)
	step_definitions.InitializeAPISteps(ctx)
	step_definitions.InitializeHookSteps(ctx) // This includes BeforeScenario and the server start step
}
