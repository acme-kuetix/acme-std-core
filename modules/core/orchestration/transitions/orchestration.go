package transitions

import (
	"fmt"

	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
)

const maxSubFlowDepth = 5

// orchestrationTransitions exposes WSL-callable sub-workflow invocation.
// Wraps engine.workflow.ExecuteWithRunnerAndSharedContext so WSL workflows
// can run feature/solution/workflow sub-flows with a shared context.
// The engine's built-in `action feature <name>` dispatch is broken (the
// worker's command-split expects a `:`-delimited state name, not the
// space-separated QName the parser produces). This transition bypasses
// that broken path by calling ExecuteWithRunnerAndSharedContext directly
// and returning a FlowStepResult.
type orchestrationTransitions struct {
	workflow.BaseServiceTransition
}

func NewOrchestrationTransitions() interfaces.ServiceTransitions {
	return &orchestrationTransitions{}
}

// RunFeature executes a feature flow by name with the current shared
// WorkerSessionContext. Returns FlowStepResult with the feature's response.
// On error, r.Success=false and r.Response holds the error message.
// WSL: action core/orchestration.RunFeature(name: "my_feature")
func (t *orchestrationTransitions) RunFeature(p *workflow.WorkerSessionContext, name string) (r domain.FlowStepResult) {
	return runSubFlow(p, "feature", name)
}

// RunWorkflow executes a workflow flow by name with the current shared
// WorkerSessionContext. Returns FlowStepResult with the workflow's response.
// WSL: action core/orchestration.RunWorkflow(name: "my_workflow")
func (t *orchestrationTransitions) RunWorkflow(p *workflow.WorkerSessionContext, name string) (r domain.FlowStepResult) {
	return runSubFlow(p, "workflow", name)
}

func runSubFlow(p *workflow.WorkerSessionContext, flowType, name string) (r domain.FlowStepResult) {
	depth := subFlowDepth(p)
	if depth >= maxSubFlowDepth {
		r.Success = false
		r.StatusCode = 500
		r.Error = fmt.Errorf("max sub-flow depth (%d) exceeded", maxSubFlowDepth)
		r.Response = map[string]interface{}{
			"message": fmt.Sprintf("%s '%s' failed: max depth %d exceeded (possible recursion)", flowType, name, maxSubFlowDepth),
			"code":    "max_depth_exceeded",
		}
		return
	}
	subFlowEnter(p)
	defer subFlowExit(p)

	app := p.Engine.GetApplication()
	wfConfig := domain.WorkflowConfigItem{
		Name:          name,
		Path:          app.Env.Config.Application.WorkflowsPath,
		Amount:        1,
		Retry:         1,
		RestartPolicy: "stop",
	}
	var runner workflow.Runner
	switch flowType {
	case "feature":
		runner = workflow.NewFeatureRunner(wfConfig, app)
	default:
		runner = workflow.NewWorkflowRunner(wfConfig, app)
	}
	responses, err := runner.RunWithSharedContext(p, name)
	if err != nil {
		r.Success = false
		r.StatusCode = 500
		r.Error = err
		r.Response = map[string]interface{}{
			"message": fmt.Sprintf("%s '%s' failed: %v", flowType, name, err),
			"code":    fmt.Sprintf("%s_failed", flowType),
		}
		return
	}
	for _, resp := range responses {
		if resp != nil && resp.IsError() {
			r.Success = false
			r.StatusCode = resp.StatusCode
			if r.StatusCode == 0 {
				r.StatusCode = 500
			}
			if resp.Error != nil && len(resp.Error.Issues) > 0 {
				r.Error = fmt.Errorf("%v", resp.Error.Issues[0])
			}
			r.Response = map[string]interface{}{
				"message": fmt.Sprintf("%s '%s' returned error", flowType, name),
				"code":    fmt.Sprintf("%s_failed", flowType),
			}
			return
		}
	}
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{
		"responses": responses,
		"name":      name,
		"type":      flowType,
	}
	return
}

const subFlowDepthKey = "__subflow_depth__"

func subFlowDepth(p *workflow.WorkerSessionContext) int {
	v := p.Value(subFlowDepthKey)
	if n, ok := v.(int); ok {
		return n
	}
	return 0
}

func subFlowEnter(p *workflow.WorkerSessionContext) {
	p.SetValue(subFlowDepthKey, subFlowDepth(p)+1)
}

func subFlowExit(p *workflow.WorkerSessionContext) {
	p.SetValue(subFlowDepthKey, subFlowDepth(p)-1)
}
