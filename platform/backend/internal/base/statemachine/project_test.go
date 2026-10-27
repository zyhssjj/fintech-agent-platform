package statemachine

import "testing"

func TestValidTransition(t *testing.T) {
	if !CanTransitionProject(ProjectDraft, ProjectScreening) {
		t.Fatalf("expected %s -> %s to be valid", ProjectDraft, ProjectScreening)
	}
}

func TestInvalidTransition(t *testing.T) {
	if CanTransitionProject(ProjectDraft, ProjectInvested) {
		t.Fatalf("expected %s -> %s to be invalid", ProjectDraft, ProjectInvested)
	}
}

func TestAllTransitions(t *testing.T) {
	allowed := map[string]map[string]bool{}
	for from, tos := range projectTransitions {
		allowed[from] = map[string]bool{}
		for _, to := range tos {
			allowed[from][to] = true
		}
	}

	states := []string{
		ProjectDraft,
		ProjectScreening,
		ProjectDueDiligence,
		ProjectDecisionPending,
		ProjectApproved,
		ProjectRejected,
		ProjectInvested,
		ProjectMonitoring,
		ProjectExitPreparing,
		ProjectExited,
	}

	for _, from := range states {
		for _, to := range states {
			got := CanTransitionProject(from, to)
			want := allowed[from][to]
			if got != want {
				t.Fatalf("transition %s -> %s = %v, want %v", from, to, got, want)
			}
		}
	}
}
