package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/nektos/act/pkg/common"
	"github.com/nektos/act/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"
)

type stepActionLocalMocks struct {
	mock.Mock
}

func (salm *stepActionLocalMocks) runAction(step actionStep, actionDir string, remoteAction *remoteAction) common.Executor {
	args := salm.Called(step, actionDir, remoteAction)
	return args.Get(0).(func(context.Context) error)
}

func (salm *stepActionLocalMocks) readAction(ctx context.Context, step *model.Step, actionDir string, actionPath string, readFile actionYamlReader, writeFile fileWriter) (*model.Action, error) {
	args := salm.Called(step, actionDir, actionPath, readFile, writeFile)
	return args.Get(0).(*model.Action), args.Error(1)
}

func TestStepActionLocalTest(t *testing.T) {
	ctx := context.Background()

	cm := &containerMock{}
	salm := &stepActionLocalMocks{}

	sal := &stepActionLocal{
		readAction: salm.readAction,
		runAction:  salm.runAction,
		RunContext: &RunContext{
			StepResults: map[string]*model.StepResult{},
			ExprEval:    &expressionEvaluator{},
			Config: &Config{
				Workdir: "/tmp",
			},
			Run: &model.Run{
				JobID: "1",
				Workflow: &model.Workflow{
					Jobs: map[string]*model.Job{
						"1": {
							Defaults: model.Defaults{
								Run: model.RunDefaults{
									Shell: "bash",
								},
							},
						},
					},
				},
			},
			JobContainer: cm,
		},
		Step: &model.Step{
			ID:   "1",
			Uses: "./path/to/action",
		},
	}

	salm.On("readAction", sal.Step, "/tmp/path/to/action", "", mock.Anything, mock.Anything).
		Return(&model.Action{}, nil)

	cm.On("UpdateFromImageEnv", mock.AnythingOfType("*map[string]string")).Return(func(ctx context.Context) error {
		return nil
	})

	cm.On("UpdateFromEnv", "/var/run/act/workflow/envs.txt", mock.AnythingOfType("*map[string]string")).Return(func(ctx context.Context) error {
		return nil
	})

	cm.On("UpdateFromPath", mock.AnythingOfType("*map[string]string")).Return(func(ctx context.Context) error {
		return nil
	})

	salm.On("runAction", sal, "/tmp/path/to/action", (*remoteAction)(nil)).Return(func(ctx context.Context) error {
		return nil
	})

	err := sal.pre()(ctx)
	assert.Nil(t, err)

	err = sal.main()(ctx)
	assert.Nil(t, err)

	cm.AssertExpectations(t)
	salm.AssertExpectations(t)
}

func TestStepActionLocalPre(t *testing.T) {
	cm := &containerMock{}
	salm := &stepActionLocalMocks{}

	ctx := context.Background()

	sal := &stepActionLocal{
		readAction: salm.readAction,
		RunContext: &RunContext{
			StepResults: map[string]*model.StepResult{},
			ExprEval:    &expressionEvaluator{},
			Config: &Config{
				Workdir: "/tmp",
			},
			Run: &model.Run{
				JobID: "1",
				Workflow: &model.Workflow{
					Jobs: map[string]*model.Job{
						"1": {
							Defaults: model.Defaults{
								Run: model.RunDefaults{
									Shell: "bash",
								},
							},
						},
					},
				},
			},
			JobContainer: cm,
		},
		Step: &model.Step{
			ID:   "1",
			Uses: "./path/to/action",
		},
	}

	salm.On("readAction", sal.Step, "/tmp/path/to/action", "", mock.Anything, mock.Anything).
		Return(&model.Action{}, nil)

	err := sal.pre()(ctx)
	assert.Nil(t, err)

	cm.AssertExpectations(t)
	salm.AssertExpectations(t)
}

func TestStepActionLocalPost(t *testing.T) {
	table := []struct {
		name                   string
		stepModel              *model.Step
		actionModel            *model.Action
		initialStepResults     map[string]*model.StepResult
		expectedPostStepResult *model.StepResult
		err                    error
		mocks                  struct {
			env  bool
			exec bool
		}
	}{
		{
			name: "main-success",
			stepModel: &model.Step{
				ID:   "step",
				Uses: "./local/action",
			},
			actionModel: &model.Action{
				Runs: model.ActionRuns{
					Using:  "node16",
					Post:   "post.js",
					PostIf: "always()",
				},
			},
			initialStepResults: map[string]*model.StepResult{
				"step": {
					Conclusion: model.StepStatusSuccess,
					Outcome:    model.StepStatusSuccess,
					Outputs:    map[string]string{},
				},
			},
			expectedPostStepResult: &model.StepResult{
				Conclusion: model.StepStatusSuccess,
				Outcome:    model.StepStatusSuccess,
				Outputs:    map[string]string{},
			},
			mocks: struct {
				env  bool
				exec bool
			}{
				env:  true,
				exec: true,
			},
		},
		{
			name: "main-failed",
			stepModel: &model.Step{
				ID:   "step",
				Uses: "./local/action",
			},
			actionModel: &model.Action{
				Runs: model.ActionRuns{
					Using:  "node16",
					Post:   "post.js",
					PostIf: "always()",
				},
			},
			initialStepResults: map[string]*model.StepResult{
				"step": {
					Conclusion: model.StepStatusFailure,
					Outcome:    model.StepStatusFailure,
					Outputs:    map[string]string{},
				},
			},
			expectedPostStepResult: &model.StepResult{
				Conclusion: model.StepStatusSuccess,
				Outcome:    model.StepStatusSuccess,
				Outputs:    map[string]string{},
			},
			mocks: struct {
				env  bool
				exec bool
			}{
				env:  true,
				exec: true,
			},
		},
		{
			name: "skip-if-failed",
			stepModel: &model.Step{
				ID:   "step",
				Uses: "./local/action",
			},
			actionModel: &model.Action{
				Runs: model.ActionRuns{
					Using:  "node16",
					Post:   "post.js",
					PostIf: "success()",
				},
			},
			initialStepResults: map[string]*model.StepResult{
				"step": {
					Conclusion: model.StepStatusFailure,
					Outcome:    model.StepStatusFailure,
					Outputs:    map[string]string{},
				},
			},
			expectedPostStepResult: &model.StepResult{
				Conclusion: model.StepStatusSkipped,
				Outcome:    model.StepStatusSkipped,
				Outputs:    map[string]string{},
			},
			mocks: struct {
				env  bool
				exec bool
			}{
				env:  true,
				exec: false,
			},
		},
		{
			name: "skip-if-main-skipped",
			stepModel: &model.Step{
				ID:   "step",
				If:   yaml.Node{Value: "failure()"},
				Uses: "./local/action",
			},
			actionModel: &model.Action{
				Runs: model.ActionRuns{
					Using:  "node16",
					Post:   "post.js",
					PostIf: "always()",
				},
			},
			initialStepResults: map[string]*model.StepResult{
				"step": {
					Conclusion: model.StepStatusSkipped,
					Outcome:    model.StepStatusSkipped,
					Outputs:    map[string]string{},
				},
			},
			expectedPostStepResult: nil,
			mocks: struct {
				env  bool
				exec bool
			}{
				env:  false,
				exec: false,
			},
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			cm := &containerMock{}

			sal := &stepActionLocal{
				env: map[string]string{},
				RunContext: &RunContext{
					Config: &Config{
						GitHubInstance: "https://github.com",
					},
					JobContainer: cm,
					Run: &model.Run{
						JobID: "1",
						Workflow: &model.Workflow{
							Jobs: map[string]*model.Job{
								"1": {},
							},
						},
					},
					StepResults: tt.initialStepResults,
				},
				Step:   tt.stepModel,
				action: tt.actionModel,
			}

			if tt.mocks.env {
				cm.On("UpdateFromImageEnv", &sal.env).Return(func(ctx context.Context) error { return nil })
				cm.On("UpdateFromEnv", "/var/run/act/workflow/envs.txt", &sal.env).Return(func(ctx context.Context) error { return nil })
				cm.On("UpdateFromPath", &sal.env).Return(func(ctx context.Context) error { return nil })
			}
			if tt.mocks.exec {
				suffixMatcher := func(suffix string) interface{} {
					return mock.MatchedBy(func(array []string) bool {
						return strings.HasSuffix(array[1], suffix)
					})
				}
				cm.On("Exec", suffixMatcher("pkg/runner/local/action/post.js"), sal.env, "", "").Return(func(ctx context.Context) error { return tt.err })
			}

			err := sal.post()(ctx)

			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.expectedPostStepResult, sal.RunContext.StepResults["post-step"])
			cm.AssertExpectations(t)
		})
	}
}
