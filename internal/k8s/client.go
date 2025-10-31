package k8s

import (
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client wraps the Kubernetes client for pod operations.
type Client struct {
	clientset kubernetes.Interface
	namespace string
}

// NewInClusterClient creates a new Kubernetes client using in-cluster configuration.
// Returns an error if not running inside a Kubernetes cluster or if configuration fails.
func NewInClusterClient() (*Client, error) {
	// Get in-cluster config using service account token
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Get current namespace from service account
	namespace, err := getCurrentNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get current namespace: %w", err)
	}

	return &Client{
		clientset: clientset,
		namespace: namespace,
	}, nil
}

// getCurrentNamespace reads the namespace from the service account mount.
func getCurrentNamespace() (string, error) {
	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read namespace file: %w", err)
	}

	// Trim whitespace including trailing newline that's typically present in the file
	return strings.TrimSpace(string(data)), nil
}

// PodInfo contains essential information about a discovered pod.
type PodInfo struct {
	Name             string
	IP               string
	AvailabilityZone string
}

// DiscoverPods finds all pods in the current namespace matching the given label selector.
// It extracts the pod name, IP, and availability zone from each pod.
// This method implements the interface expected by the infrabin package.
func (c *Client) DiscoverPods(ctx context.Context, labelSelector string) ([]PodInfo, error) {
	// List pods with label selector
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Extract pod information
	var result []PodInfo
	for _, pod := range pods.Items {
		// Skip pods that are not running
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		// Skip pods without IP
		if pod.Status.PodIP == "" {
			continue
		}

		// Extract AZ - first from pod, then from node
		az, err := c.extractAZ(ctx, &pod)
		if err != nil {
			return nil, fmt.Errorf("failed to extract AZ for pod %s: %w", pod.Name, err)
		}
		if az == "unknown" {
			return nil, fmt.Errorf("pod %s has unknown availability zone (no zone label on pod or node %s)", pod.Name, pod.Spec.NodeName)
		}

		result = append(result, PodInfo{
			Name:             pod.Name,
			IP:               pod.Status.PodIP,
			AvailabilityZone: az,
		})
	}

	return result, nil
}

// extractAZ extracts the availability zone for a pod.
// It first checks the pod's node selector for AZ labels.
// If not found, it queries the node's labels.
// Returns "unknown" if AZ cannot be determined from either source.
func (c *Client) extractAZ(ctx context.Context, pod *corev1.Pod) (string, error) {
	// Try to get AZ from pod's node selector first (faster, no API call)
	// Try standard topology label
	if zone, ok := pod.Spec.NodeSelector["topology.kubernetes.io/zone"]; ok {
		return zone, nil
	}

	// Try legacy label
	if zone, ok := pod.Spec.NodeSelector["failure-domain.beta.kubernetes.io/zone"]; ok {
		return zone, nil
	}

	// If not in node selector, query the node for AZ labels
	if pod.Spec.NodeName == "" {
		return "", fmt.Errorf("pod has no node assigned")
	}

	node, err := c.clientset.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node %s: %w", pod.Spec.NodeName, err)
	}

	// Try standard topology label on node
	if zone, ok := node.Labels["topology.kubernetes.io/zone"]; ok {
		return zone, nil
	}

	// Try legacy label on node
	if zone, ok := node.Labels["failure-domain.beta.kubernetes.io/zone"]; ok {
		return zone, nil
	}

	// Could not determine AZ from pod or node
	return "", fmt.Errorf("no availability zone label found on node %s (expected topology.kubernetes.io/zone or failure-domain.beta.kubernetes.io/zone)", pod.Spec.NodeName)
}
