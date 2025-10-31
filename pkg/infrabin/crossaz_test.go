package infrabin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupPodsByAZ(t *testing.T) {
	tests := []struct {
		name           string
		pods           []K8sPodInfo
		currentPodName string
		expected       map[string]*PodList
	}{
		{
			name: "groups pods by availability zone",
			pods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.1.2", AvailabilityZone: "us-east-1a"},
				{Name: "pod-3", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
				{Name: "pod-4", IP: "10.0.3.1", AvailabilityZone: "us-east-1c"},
			},
			currentPodName: "pod-1",
			expected: map[string]*PodList{
				"us-east-1a": {PodNames: []string{"pod-1", "pod-2"}},
				"us-east-1b": {PodNames: []string{"pod-3"}},
				"us-east-1c": {PodNames: []string{"pod-4"}},
			},
		},
		{
			name: "handles empty availability zone",
			pods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.1.2", AvailabilityZone: ""},
			},
			currentPodName: "pod-1",
			expected: map[string]*PodList{
				"us-east-1a": {PodNames: []string{"pod-1"}},
				"unknown":    {PodNames: []string{"pod-2"}},
			},
		},
		{
			name:           "handles empty pod list",
			pods:           []K8sPodInfo{},
			currentPodName: "pod-1",
			expected:       map[string]*PodList{},
		},
		{
			name: "includes current pod in grouping",
			pods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
			},
			currentPodName: "pod-1",
			expected: map[string]*PodList{
				"us-east-1a": {PodNames: []string{"pod-1"}},
				"us-east-1b": {PodNames: []string{"pod-2"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupPodsByAZ(tt.pods, tt.currentPodName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterCrossAZPods(t *testing.T) {
	tests := []struct {
		name           string
		pods           []K8sPodInfo
		currentAZ      string
		currentPodName string
		expected       []K8sPodInfo
	}{
		{
			name: "filters out current pod and same AZ pods",
			pods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.1.2", AvailabilityZone: "us-east-1a"},
				{Name: "pod-3", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
				{Name: "pod-4", IP: "10.0.3.1", AvailabilityZone: "us-east-1c"},
			},
			currentAZ:      "us-east-1a",
			currentPodName: "pod-1",
			expected: []K8sPodInfo{
				{Name: "pod-3", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
				{Name: "pod-4", IP: "10.0.3.1", AvailabilityZone: "us-east-1c"},
			},
		},
		{
			name: "returns empty list when all pods in same AZ",
			pods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.1.2", AvailabilityZone: "us-east-1a"},
				{Name: "pod-3", IP: "10.0.1.3", AvailabilityZone: "us-east-1a"},
			},
			currentAZ:      "us-east-1a",
			currentPodName: "pod-1",
			expected:       nil,
		},
		{
			name: "filters out current pod even if in different AZ (should not happen)",
			pods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
			},
			currentAZ:      "us-east-1b",
			currentPodName: "pod-1",
			expected:       nil,
		},
		{
			name:           "handles empty pod list",
			pods:           []K8sPodInfo{},
			currentAZ:      "us-east-1a",
			currentPodName: "pod-1",
			expected:       nil,
		},
		{
			name: "includes pods with unknown AZ when current AZ is known",
			pods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.2.1", AvailabilityZone: "unknown"},
			},
			currentAZ:      "us-east-1a",
			currentPodName: "pod-1",
			expected: []K8sPodInfo{
				{Name: "pod-2", IP: "10.0.2.1", AvailabilityZone: "unknown"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterCrossAZPods(tt.pods, tt.currentAZ, tt.currentPodName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateSummary(t *testing.T) {
	tests := []struct {
		name           string
		allPods        []K8sPodInfo
		discoveredPods map[string]*PodList
		testResults    []*CrossAZTest
		expected       *CrossAZSummary
	}{
		{
			name: "calculates summary with successful and failed tests",
			allPods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.1.2", AvailabilityZone: "us-east-1a"},
				{Name: "pod-3", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
				{Name: "pod-4", IP: "10.0.3.1", AvailabilityZone: "us-east-1c"},
			},
			discoveredPods: map[string]*PodList{
				"us-east-1a": {PodNames: []string{"pod-1", "pod-2"}},
				"us-east-1b": {PodNames: []string{"pod-3"}},
				"us-east-1c": {PodNames: []string{"pod-4"}},
			},
			testResults: []*CrossAZTest{
				{PodName: "pod-3", Success: true},
				{PodName: "pod-4", Success: true},
			},
			expected: &CrossAZSummary{
				TotalPods:         4,
				TotalAzs:          3,
				CrossAzPodsTested: 2,
				SuccessfulTests:   2,
				FailedTests:       0,
			},
		},
		{
			name: "handles all failed tests",
			allPods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
			},
			discoveredPods: map[string]*PodList{
				"us-east-1a": {PodNames: []string{"pod-1"}},
				"us-east-1b": {PodNames: []string{"pod-2"}},
			},
			testResults: []*CrossAZTest{
				{PodName: "pod-2", Success: false},
			},
			expected: &CrossAZSummary{
				TotalPods:         2,
				TotalAzs:          2,
				CrossAzPodsTested: 1,
				SuccessfulTests:   0,
				FailedTests:       1,
			},
		},
		{
			name: "handles empty test results",
			allPods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
			},
			discoveredPods: map[string]*PodList{
				"us-east-1a": {PodNames: []string{"pod-1"}},
			},
			testResults: []*CrossAZTest{},
			expected: &CrossAZSummary{
				TotalPods:         1,
				TotalAzs:          1,
				CrossAzPodsTested: 0,
				SuccessfulTests:   0,
				FailedTests:       0,
			},
		},
		{
			name: "handles mixed success and failure",
			allPods: []K8sPodInfo{
				{Name: "pod-1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
				{Name: "pod-2", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
				{Name: "pod-3", IP: "10.0.3.1", AvailabilityZone: "us-east-1c"},
			},
			discoveredPods: map[string]*PodList{
				"us-east-1a": {PodNames: []string{"pod-1"}},
				"us-east-1b": {PodNames: []string{"pod-2"}},
				"us-east-1c": {PodNames: []string{"pod-3"}},
			},
			testResults: []*CrossAZTest{
				{PodName: "pod-2", Success: true},
				{PodName: "pod-3", Success: false},
			},
			expected: &CrossAZSummary{
				TotalPods:         3,
				TotalAzs:          3,
				CrossAzPodsTested: 2,
				SuccessfulTests:   1,
				FailedTests:       1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSummary(tt.allPods, tt.discoveredPods, tt.testResults)
			assert.Equal(t, tt.expected, result)
		})
	}
}
