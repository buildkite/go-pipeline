package pipeline

import (
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestPipelineSecretsUnmarshal(t *testing.T) {
	t.Parallel()

	// Test parsing pipeline with build-level secrets
	yamlData := `
secrets:
  - DATABASE_URL
  - API_TOKEN
steps:
  - command: echo "hello"
`

	var p Pipeline
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &p)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	if len(p.Secrets) != 2 {
		t.Fatalf("len(p.Secrets) = %d, want 2", len(p.Secrets))
	}

	want := Secrets{
		{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
		{Key: "API_TOKEN", EnvironmentVariable: "API_TOKEN"},
	}

	if diff := cmp.Diff(p.Secrets, want); diff != "" {
		t.Errorf("p.Secrets = %v, want %v", p.Secrets, want)
	}
}

func TestPipelineSecretsWithSteps(t *testing.T) {
	t.Parallel()

	// Test complete pipeline with both pipeline and step secrets
	yamlData := `
secrets:
  - DATABASE_URL
  - REDIS_URL
steps:
  - command: echo "step1"
    secrets:
      - API_TOKEN
      - DATABASE_URL
  - command: echo "step2"
`

	var p Pipeline
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &p)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	// Check pipeline secrets
	if len(p.Secrets) != 2 {
		t.Fatalf("len(p.Secrets) = %d, want 2", len(p.Secrets))
	}

	// Check steps
	if len(p.Steps) != 2 {
		t.Fatalf("len(p.Steps) = %d, want 2", len(p.Steps))
	}

	// First step should have its own secrets
	step1, ok := p.Steps[0].(*CommandStep)
	if !ok {
		t.Fatalf("p.Steps[0] is not a CommandStep")
	}
	if len(step1.Secrets) != 2 {
		t.Fatalf("len(step1.Secrets) = %d, want 2", len(step1.Secrets))
	}

	// Second step should have no secrets initially
	step2, ok := p.Steps[1].(*CommandStep)
	if !ok {
		t.Fatalf("p.Steps[1] is not a CommandStep")
	}
	if len(step2.Secrets) != 0 {
		t.Fatalf("len(step2.Secrets) = %d, want 0", len(step2.Secrets))
	}

	// Test merging for both steps
	step1.MergeSecretsFromPipeline(p.Secrets)
	step2.MergeSecretsFromPipeline(p.Secrets)

	// Step1 should have 3 secrets (2 from step + 1 from pipeline not overridden)
	if len(step1.Secrets) != 3 {
		t.Fatalf("len(step1.Secrets after merge) = %d, want 3", len(step1.Secrets))
	}

	// Step2 should have 2 secrets (both from pipeline)
	if len(step2.Secrets) != 2 {
		t.Fatalf("len(step2.Secrets after merge) = %d, want 2", len(step2.Secrets))
	}
}
