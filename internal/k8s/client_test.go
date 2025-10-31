package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestExtractAZFromPodNodeSelector(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected string
	}{
		{
			name: "standard topology label in node selector",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"topology.kubernetes.io/zone": "us-east-1a",
					},
				},
			},
			expected: "us-east-1a",
		},
		{
			name: "legacy label in node selector",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"failure-domain.beta.kubernetes.io/zone": "us-west-2b",
					},
				},
			},
			expected: "us-west-2b",
		},
	}

	// Create fake client for testing
	client := &Client{
		clientset: fake.NewSimpleClientset(),
		namespace: "default",
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.extractAZ(ctx, tt.pod)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractAZFromNode(t *testing.T) {
	// Create a fake node with AZ label
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
			Labels: map[string]string{
				"topology.kubernetes.io/zone": "us-east-1c",
			},
		},
	}

	// Create fake clientset with the node
	fakeClient := fake.NewSimpleClientset(node)

	client := &Client{
		clientset: fakeClient,
		namespace: "default",
	}

	// Create pod without node selector but with node name
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			NodeName:     "test-node",
			NodeSelector: map[string]string{},
		},
	}

	ctx := context.Background()
	result, err := client.extractAZ(ctx, pod)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "us-east-1c" {
		t.Errorf("Expected us-east-1c, got %s", result)
	}
}

func TestExtractAZNoNodeName(t *testing.T) {
	client := &Client{
		clientset: fake.NewSimpleClientset(),
		namespace: "default",
	}

	// Pod with no node selector and no node name
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			NodeName:     "",
			NodeSelector: map[string]string{},
		},
	}

	ctx := context.Background()
	_, err := client.extractAZ(ctx, pod)
	if err == nil {
		t.Fatal("Expected error for pod with no node assigned, got nil")
	}
}

func TestExtractAZNodeWithoutZoneLabel(t *testing.T) {
	// Create a fake node without AZ label
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-node",
			Labels: map[string]string{
				// No zone label
				"kubernetes.io/hostname": "test-node",
			},
		},
	}

	// Create fake clientset with the node
	fakeClient := fake.NewSimpleClientset(node)

	client := &Client{
		clientset: fakeClient,
		namespace: "default",
	}

	// Create pod without node selector but with node name
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			NodeName:     "test-node",
			NodeSelector: map[string]string{},
		},
	}

	ctx := context.Background()
	_, err := client.extractAZ(ctx, pod)
	if err == nil {
		t.Fatal("Expected error for node without zone label, got nil")
	}
}

func TestDiscoverPodsFiltersNonRunning(t *testing.T) {
	// Create test pods with different phases
	pendingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"topology.kubernetes.io/zone": "us-east-1a",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			PodIP: "10.0.1.1",
		},
	}

	runningPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"topology.kubernetes.io/zone": "us-east-1b",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.2.1",
		},
	}

	// Create fake clientset with pods
	fakeClient := fake.NewSimpleClientset(pendingPod, runningPod)

	client := &Client{
		clientset: fakeClient,
		namespace: "default",
	}

	ctx := context.Background()
	pods, err := client.DiscoverPods(ctx, "app=test")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should only include running pod
	if len(pods) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(pods))
	}

	if len(pods) > 0 && pods[0].Name != "running-pod" {
		t.Errorf("Expected running-pod, got %s", pods[0].Name)
	}
}
