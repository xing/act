package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/assert"

	"github.com/nektos/act/pkg/common"
	"github.com/nektos/act/pkg/model"
)

var (
	baseImage = "node:16-buster-slim"
	platforms map[string]string
	logLevel  = log.DebugLevel
	workdir   = "testdata"
	secrets   map[string]string
)

func init() {
	if p := os.Getenv("ACT_TEST_IMAGE"); p != "" {
		baseImage = p
	}

	platforms = map[string]string{
		"ubuntu-latest": baseImage,
	}

	if l := os.Getenv("ACT_TEST_LOG_LEVEL"); l != "" {
		if lvl, err := log.ParseLevel(l); err == nil {
			logLevel = lvl
		}
	}

	if wd, err := filepath.Abs(workdir); err == nil {
		workdir = wd
	}

	secrets = map[string]string{}
}

func TestGraphEvent(t *testing.T) {
	planner, err := model.NewWorkflowPlanner("testdata/basic", true)
	assert.Nil(t, err)

	plan := planner.PlanEvent("push")
	assert.Nil(t, err)
	assert.Equal(t, len(plan.Stages), 3, "stages")
	assert.Equal(t, len(plan.Stages[0].Runs), 1, "stage0.runs")
	assert.Equal(t, len(plan.Stages[1].Runs), 1, "stage1.runs")
	assert.Equal(t, len(plan.Stages[2].Runs), 1, "stage2.runs")
	assert.Equal(t, plan.Stages[0].Runs[0].JobID, "check", "jobid")
	assert.Equal(t, plan.Stages[1].Runs[0].JobID, "build", "jobid")
	assert.Equal(t, plan.Stages[2].Runs[0].JobID, "test", "jobid")

	plan = planner.PlanEvent("release")
	assert.Equal(t, len(plan.Stages), 0, "stages")
}

type TestJobFileInfo struct {
	workdir      string
	workflowPath string
	eventName    string
	errorMessage string
	platforms    map[string]string
	secrets      map[string]string
}

func (j *TestJobFileInfo) runTest(ctx context.Context, t *testing.T, cfg *Config) {
	fmt.Printf("::group::%s\n", j.workflowPath)

	log.SetLevel(logLevel)

	workdir, err := filepath.Abs(j.workdir)
	assert.Nil(t, err, workdir)

	fullWorkflowPath := filepath.Join(workdir, j.workflowPath)
	runnerConfig := &Config{
		Workdir:               workdir,
		BindWorkdir:           false,
		EventName:             j.eventName,
		EventPath:             cfg.EventPath,
		Platforms:             j.platforms,
		ReuseContainers:       false,
		Env:                   cfg.Env,
		Secrets:               cfg.Secrets,
		GitHubInstance:        "github.com",
		ContainerArchitecture: cfg.ContainerArchitecture,
	}

	runner, err := New(runnerConfig)
	assert.Nil(t, err, j.workflowPath)

	planner, err := model.NewWorkflowPlanner(fullWorkflowPath, true)
	assert.Nil(t, err, fullWorkflowPath)

	plan := planner.PlanEvent(j.eventName)

	err = runner.NewPlanExecutor(plan)(ctx)
	if j.errorMessage == "" {
		assert.Nil(t, err, fullWorkflowPath)
	} else {
		assert.Error(t, err, j.errorMessage)
	}

	fmt.Println("::endgroup::")
}

func TestRunEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	tables := []TestJobFileInfo{
		// Shells
		{workdir, "shells/defaults", "push", "", platforms, secrets},
		// TODO: figure out why it fails
		// {workdir, "shells/custom", "push", "", map[string]string{"ubuntu-latest": "catthehacker/ubuntu:pwsh-latest"}, }, // custom image with pwsh
		{workdir, "shells/pwsh", "push", "", map[string]string{"ubuntu-latest": "catthehacker/ubuntu:pwsh-latest"}, secrets}, // custom image with pwsh
		{workdir, "shells/bash", "push", "", platforms, secrets},
		{workdir, "shells/python", "push", "", map[string]string{"ubuntu-latest": "node:16-buster"}, secrets}, // slim doesn't have python
		{workdir, "shells/sh", "push", "", platforms, secrets},

		// Local action
		{workdir, "local-action-docker-url", "push", "", platforms, secrets},
		{workdir, "local-action-dockerfile", "push", "", platforms, secrets},
		{workdir, "local-action-via-composite-dockerfile", "push", "", platforms, secrets},
		{workdir, "local-action-js", "push", "", platforms, secrets},

		// Uses
		{workdir, "uses-composite", "push", "", platforms, secrets},
		{workdir, "uses-composite-with-error", "push", "Job 'failing-composite-action' failed", platforms, secrets},
		{workdir, "uses-nested-composite", "push", "", platforms, secrets},
		{workdir, "remote-action-composite-js-pre-with-defaults", "push", "", platforms, secrets},
		{workdir, "uses-workflow", "push", "reusable workflows are currently not supported (see https://github.com/nektos/act/issues/826 for updates)", platforms, secrets},
		{workdir, "uses-workflow", "pull_request", "", platforms, map[string]string{"secret": "keep_it_private"}},
		{workdir, "uses-docker-url", "push", "", platforms, secrets},
		{workdir, "act-composite-env-test", "push", "", platforms, secrets},

		// Eval
		{workdir, "evalmatrix", "push", "", platforms, secrets},
		{workdir, "evalmatrixneeds", "push", "", platforms, secrets},
		{workdir, "evalmatrixneeds2", "push", "", platforms, secrets},
		{workdir, "evalmatrix-merge-map", "push", "", platforms, secrets},
		{workdir, "evalmatrix-merge-array", "push", "", platforms, secrets},
		{workdir, "issue-1195", "push", "", platforms, secrets},

		{workdir, "basic", "push", "", platforms, secrets},
		{workdir, "fail", "push", "exit with `FAILURE`: 1", platforms, secrets},
		{workdir, "runs-on", "push", "", platforms, secrets},
		{workdir, "checkout", "push", "", platforms, secrets},
		{workdir, "job-container", "push", "", platforms, secrets},
		{workdir, "job-container-non-root", "push", "", platforms, secrets},
		{workdir, "job-container-invalid-credentials", "push", "failed to handle credentials: failed to interpolate container.credentials.password", platforms, secrets},
		{workdir, "container-hostname", "push", "", platforms, secrets},
		{workdir, "remote-action-docker", "push", "", platforms, secrets},
		{workdir, "remote-action-js", "push", "", platforms, secrets},
		{workdir, "remote-action-js", "push", "", map[string]string{"ubuntu-latest": "catthehacker/ubuntu:runner-latest"}, secrets}, // Test if this works with non root container
		{workdir, "matrix", "push", "", platforms, secrets},
		{workdir, "matrix-include-exclude", "push", "", platforms, secrets},
		{workdir, "commands", "push", "", platforms, secrets},
		{workdir, "workdir", "push", "", platforms, secrets},
		{workdir, "defaults-run", "push", "", platforms, secrets},
		{workdir, "composite-fail-with-output", "push", "", platforms, secrets},
		{workdir, "issue-597", "push", "", platforms, secrets},
		{workdir, "issue-598", "push", "", platforms, secrets},
		{workdir, "if-env-act", "push", "", platforms, secrets},
		{workdir, "env-and-path", "push", "", platforms, secrets},
		{workdir, "environment-files", "push", "", platforms, secrets},
		{workdir, "GITHUB_STATE", "push", "", platforms, secrets},
		{workdir, "environment-files-parser-bug", "push", "", platforms, secrets},
		{workdir, "non-existent-action", "push", "Job 'nopanic' failed", platforms, secrets},
		{workdir, "outputs", "push", "", platforms, secrets},
		{workdir, "networking", "push", "", platforms, secrets},
		{workdir, "steps-context/conclusion", "push", "", platforms, secrets},
		{workdir, "steps-context/outcome", "push", "", platforms, secrets},
		{workdir, "job-status-check", "push", "job 'fail' failed", platforms, secrets},
		{workdir, "if-expressions", "push", "Job 'mytest' failed", platforms, secrets},
		{workdir, "actions-environment-and-context-tests", "push", "", platforms, secrets},
		{workdir, "uses-action-with-pre-and-post-step", "push", "", platforms, secrets},
		{workdir, "evalenv", "push", "", platforms, secrets},
		{workdir, "ensure-post-steps", "push", "Job 'second-post-step-should-fail' failed", platforms, secrets},
		{workdir, "workflow_dispatch", "workflow_dispatch", "", platforms, secrets},
		{workdir, "workflow_dispatch_no_inputs_mapping", "workflow_dispatch", "", platforms, secrets},
		{workdir, "workflow_dispatch-scalar", "workflow_dispatch", "", platforms, secrets},
		{workdir, "workflow_dispatch-scalar-composite-action", "workflow_dispatch", "", platforms, secrets},
		{workdir, "job-needs-context-contains-result", "push", "", platforms, secrets},
		{"../model/testdata", "strategy", "push", "", platforms, secrets}, // TODO: move all testdata into pkg so we can validate it with planner and runner
		// {"testdata", "issue-228", "push", "", platforms, }, // TODO [igni]: Remove this once everything passes
		{"../model/testdata", "container-volumes", "push", "", platforms, secrets},
	}

	for _, table := range tables {
		t.Run(table.workflowPath, func(t *testing.T) {
			config := &Config{
				Secrets: table.secrets,
			}

			eventFile := filepath.Join(workdir, table.workflowPath, "event.json")
			if _, err := os.Stat(eventFile); err == nil {
				config.EventPath = eventFile
			}

			table.runTest(ctx, t, config)
		})
	}
}

func TestRunEventHostEnvironment(t *testing.T) {
	// if testing.Short() {
	t.Skip("skipping HostEnvironment test")
	// }

	ctx := context.Background()

	tables := []TestJobFileInfo{}

	if runtime.GOOS == "linux" {
		platforms := map[string]string{
			"ubuntu-latest": "-self-hosted",
		}

		tables = append(tables, []TestJobFileInfo{
			// Shells
			{workdir, "shells/defaults", "push", "", platforms, secrets},
			{workdir, "shells/pwsh", "push", "", platforms, secrets},
			{workdir, "shells/bash", "push", "", platforms, secrets},
			{workdir, "shells/python", "push", "", platforms, secrets},
			{workdir, "shells/sh", "push", "", platforms, secrets},

			// Local action
			{workdir, "local-action-js", "push", "", platforms, secrets},

			// Uses
			{workdir, "uses-composite", "push", "", platforms, secrets},
			{workdir, "uses-composite-with-error", "push", "Job 'failing-composite-action' failed", platforms, secrets},
			{workdir, "uses-nested-composite", "push", "", platforms, secrets},
			{workdir, "act-composite-env-test", "push", "", platforms, secrets},

			// Eval
			{workdir, "evalmatrix", "push", "", platforms, secrets},
			{workdir, "evalmatrixneeds", "push", "", platforms, secrets},
			{workdir, "evalmatrixneeds2", "push", "", platforms, secrets},
			{workdir, "evalmatrix-merge-map", "push", "", platforms, secrets},
			{workdir, "evalmatrix-merge-array", "push", "", platforms, secrets},
			{workdir, "issue-1195", "push", "", platforms, secrets},

			{workdir, "fail", "push", "exit with `FAILURE`: 1", platforms, secrets},
			{workdir, "runs-on", "push", "", platforms, secrets},
			{workdir, "checkout", "push", "", platforms, secrets},
			{workdir, "remote-action-js", "push", "", platforms, secrets},
			{workdir, "matrix", "push", "", platforms, secrets},
			{workdir, "matrix-include-exclude", "push", "", platforms, secrets},
			{workdir, "commands", "push", "", platforms, secrets},
			{workdir, "defaults-run", "push", "", platforms, secrets},
			{workdir, "composite-fail-with-output", "push", "", platforms, secrets},
			{workdir, "issue-597", "push", "", platforms, secrets},
			{workdir, "issue-598", "push", "", platforms, secrets},
			{workdir, "if-env-act", "push", "", platforms, secrets},
			{workdir, "env-and-path", "push", "", platforms, secrets},
			{workdir, "non-existent-action", "push", "Job 'nopanic' failed", platforms, secrets},
			{workdir, "outputs", "push", "", platforms, secrets},
			{workdir, "steps-context/conclusion", "push", "", platforms, secrets},
			{workdir, "steps-context/outcome", "push", "", platforms, secrets},
			{workdir, "job-status-check", "push", "job 'fail' failed", platforms, secrets},
			{workdir, "if-expressions", "push", "Job 'mytest' failed", platforms, secrets},
			{workdir, "uses-action-with-pre-and-post-step", "push", "", platforms, secrets},
			{workdir, "evalenv", "push", "", platforms, secrets},
			{workdir, "ensure-post-steps", "push", "Job 'second-post-step-should-fail' failed", platforms, secrets},
		}...)
	}
	if runtime.GOOS == "windows" {
		platforms := map[string]string{
			"windows-latest": "-self-hosted",
		}

		tables = append(tables, []TestJobFileInfo{
			{workdir, "windows-prepend-path", "push", "", platforms, secrets},
			{workdir, "windows-add-env", "push", "", platforms, secrets},
		}...)
	} else {
		platforms := map[string]string{
			"self-hosted": "-self-hosted",
		}

		tables = append(tables, []TestJobFileInfo{
			{workdir, "nix-prepend-path", "push", "", platforms, secrets},
			{workdir, "inputs-via-env-context", "push", "", platforms, secrets},
		}...)
	}

	for _, table := range tables {
		t.Run(table.workflowPath, func(t *testing.T) {
			table.runTest(ctx, t, &Config{})
		})
	}
}

func TestDryrunEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := common.WithDryrun(context.Background(), true)

	tables := []TestJobFileInfo{
		// Shells
		{workdir, "shells/defaults", "push", "", platforms, secrets},
		{workdir, "shells/pwsh", "push", "", map[string]string{"ubuntu-latest": "catthehacker/ubuntu:pwsh-latest"}, secrets}, // custom image with pwsh
		{workdir, "shells/bash", "push", "", platforms, secrets},
		{workdir, "shells/python", "push", "", map[string]string{"ubuntu-latest": "node:16-buster"}, secrets}, // slim doesn't have python
		{workdir, "shells/sh", "push", "", platforms, secrets},

		// Local action
		{workdir, "local-action-docker-url", "push", "", platforms, secrets},
		{workdir, "local-action-dockerfile", "push", "", platforms, secrets},
		{workdir, "local-action-via-composite-dockerfile", "push", "", platforms, secrets},
		{workdir, "local-action-js", "push", "", platforms, secrets},
	}

	for _, table := range tables {
		t.Run(table.workflowPath, func(t *testing.T) {
			table.runTest(ctx, t, &Config{})
		})
	}
}

func TestRunDifferentArchitecture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tjfi := TestJobFileInfo{
		workdir:      workdir,
		workflowPath: "basic",
		eventName:    "push",
		errorMessage: "",
		platforms:    platforms,
	}

	tjfi.runTest(context.Background(), t, &Config{ContainerArchitecture: "linux/arm64"})
}

type maskJobLoggerFactory struct {
	Output bytes.Buffer
}

func (f *maskJobLoggerFactory) WithJobLogger() *log.Logger {
	logger := log.New()
	logger.SetOutput(io.MultiWriter(&f.Output, os.Stdout))
	logger.SetLevel(log.DebugLevel)
	return logger
}

func TestMaskValues(t *testing.T) {
	assertNoSecret := func(text string, secret string) {
		index := strings.Index(text, "composite secret")
		if index > -1 {
			fmt.Printf("\nFound Secret in the given text:\n%s\n", text)
		}
		assert.False(t, strings.Contains(text, "composite secret"))
	}

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	log.SetLevel(log.DebugLevel)

	tjfi := TestJobFileInfo{
		workdir:      workdir,
		workflowPath: "mask-values",
		eventName:    "push",
		errorMessage: "",
		platforms:    platforms,
	}

	logger := &maskJobLoggerFactory{}
	tjfi.runTest(WithJobLoggerFactory(common.WithLogger(context.Background(), logger.WithJobLogger()), logger), t, &Config{})
	output := logger.Output.String()

	assertNoSecret(output, "secret value")
	assertNoSecret(output, "YWJjCg==")
}

func TestRunEventSecrets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	workflowPath := "secrets"

	tjfi := TestJobFileInfo{
		workdir:      workdir,
		workflowPath: workflowPath,
		eventName:    "push",
		errorMessage: "",
		platforms:    platforms,
	}

	env, err := godotenv.Read(filepath.Join(workdir, workflowPath, ".env"))
	assert.NoError(t, err, "Failed to read .env")
	secrets, _ := godotenv.Read(filepath.Join(workdir, workflowPath, ".secrets"))
	assert.NoError(t, err, "Failed to read .secrets")

	tjfi.runTest(context.Background(), t, &Config{Secrets: secrets, Env: env})
}

func TestRunEventPullRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	workflowPath := "pull-request"

	tjfi := TestJobFileInfo{
		workdir:      workdir,
		workflowPath: workflowPath,
		eventName:    "pull_request",
		errorMessage: "",
		platforms:    platforms,
	}

	tjfi.runTest(context.Background(), t, &Config{EventPath: filepath.Join(workdir, workflowPath, "event.json")})
}
