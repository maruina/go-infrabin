# Cross-AZ Connectivity Endpoint Implementation Plan

## Overview

This plan details the implementation steps for the `/crossaz` endpoint feature based on the approved specification. The implementation will be done incrementally to enable testing at each stage.

## Implementation Phases

### Phase 1: Dependencies and Proto Definitions

#### 1.1 Add Go Dependencies

**File**: `go.mod`

Add the following dependencies:
```go
k8s.io/client-go v0.31.3
k8s.io/api v0.31.3
k8s.io/apimachinery v0.31.3
```

**Command**: `go get k8s.io/client-go@v0.31.3 k8s.io/api@v0.31.3 k8s.io/apimachinery@v0.31.3`

#### 1.2 Update Proto Definitions

**File**: `proto/infrabin/infrabin.proto`

Add the following to the `Infrabin` service (after `SetReadinessStatus`):

```protobuf
// CrossAZ performs cross-availability-zone connectivity tests.
// Discovers all go-infrabin pods in different AZs and tests connectivity to them.
// Requires --enable-crossaz-endpoint flag and appropriate RBAC permissions.
rpc CrossAZ(Empty) returns (CrossAZResponse) {
    option (google.api.http) = {
        get: "/crossaz"
    };
}
```

Add the following message definitions (after `SetHealthStatusRequest`):

```protobuf
// CrossAZResponse contains the results of cross-AZ connectivity tests.
message CrossAZResponse {
    // current_az is the availability zone of the pod handling this request.
    string current_az = 1;
    // current_pod is the name of the pod handling this request.
    string current_pod = 2;
    // discovered_pods maps availability zones to lists of pod names.
    map<string, PodList> discovered_pods = 3;
    // cross_az_tests contains connectivity test results for pods in other AZs.
    repeated CrossAZTest cross_az_tests = 4;
    // summary contains aggregate statistics about the tests.
    CrossAZSummary summary = 5;
}

// PodList contains a list of pod names.
message PodList {
    repeated string pod_names = 1;
}

// CrossAZTest represents a single connectivity test to a pod in another AZ.
message CrossAZTest {
    // pod_name is the name of the target pod.
    string pod_name = 1;
    // pod_ip is the IP address of the target pod.
    string pod_ip = 2;
    // az is the availability zone of the target pod.
    string az = 3;
    // success indicates whether connectivity was successful.
    bool success = 4;
    // status_code is the HTTP status code returned (0 if connection failed).
    int32 status_code = 5;
    // duration_ms is the time taken for the request in milliseconds.
    int64 duration_ms = 6;
    // error contains error details if the test failed.
    string error = 7;
}

// CrossAZSummary contains aggregate statistics about cross-AZ tests.
message CrossAZSummary {
    // total_pods is the total number of pods discovered.
    int32 total_pods = 1;
    // total_azs is the number of unique availability zones.
    int32 total_azs = 2;
    // cross_az_pods_tested is the number of pods tested in other AZs.
    int32 cross_az_pods_tested = 3;
    // successful_tests is the number of successful connectivity tests.
    int32 successful_tests = 4;
    // failed_tests is the number of failed connectivity tests.
    int32 failed_tests = 5;
}
```

**Command**: Run `make protoc` to regenerate Go code from proto definitions.

---

### Phase 2: Kubernetes Client Package

Create a new internal package for Kubernetes operations.

#### 2.1 Create Kubernetes Client Package

**File**: `internal/k8s/client.go`

```go
package k8s

import (
    "context"
    "fmt"
    "os"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

// Client wraps the Kubernetes client for pod operations.
type Client struct {
    clientset *kubernetes.Clientset
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

    return string(data), nil
}

// PodInfo contains essential information about a discovered pod.
type PodInfo struct {
    Name             string
    IP               string
    AvailabilityZone string
}

// DiscoverPods finds all pods in the current namespace matching the given label selector.
// It extracts the pod name, IP, and availability zone from each pod.
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
            // Log error but continue with "unknown" AZ
            fmt.Printf("WARNING: failed to extract AZ for pod %s: %v\n", pod.Name, err)
            az = "unknown"
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
        return "unknown", nil
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
    return "unknown", nil
}
```

**Key Design Decisions**:
- Use `rest.InClusterConfig()` for automatic service account token authentication
- Read namespace from service account mount point (standard location)
- Filter out non-running pods to avoid testing against terminated/pending pods
- **Unified AZ extraction**: Single `extractAZ()` function that:
  1. First checks pod's node selector (fast, no API call)
  2. Falls back to querying node labels (requires nodes read permission)
  3. Returns "unknown" if AZ cannot be determined (graceful degradation)
- Handles errors gracefully and continues with "unknown" AZ on failure

---

### Phase 3: CrossAZ Implementation

#### 3.1 Add Configuration Options

**File**: `pkg/infrabin/config.go`

Add constants:
```go
const (
    // ... existing constants ...

    EnableCrossAZEndpoint = false
    CrossAZTimeout        = 3 * time.Second
    CrossAZLabelSelector  = "app.kubernetes.io/name=go-infrabin"
)
```

Update `ReadConfiguration()`:
```go
// CrossAZ endpoint configuration
viper.SetDefault("enableCrossAZEndpoint", EnableCrossAZEndpoint)
viper.SetDefault("crossAZTimeout", CrossAZTimeout)
viper.SetDefault("crossAZLabelSelector", CrossAZLabelSelector)
```

#### 3.2 Add Command-Line Flags

**File**: `internal/cmd/root.go`

Add flags in the init function:
```go
// CrossAZ endpoint flags
rootCmd.PersistentFlags().Bool("enable-crossaz-endpoint", false, "Enable the /crossaz endpoint for cross-AZ connectivity testing")
rootCmd.PersistentFlags().Duration("crossaz-timeout", 3*time.Second, "Timeout for cross-AZ connectivity tests")
rootCmd.PersistentFlags().String("crossaz-label-selector", "app.kubernetes.io/name=go-infrabin", "Label selector for discovering go-infrabin pods")

// Bind flags to viper
_ = viper.BindPFlag("enableCrossAZEndpoint", rootCmd.PersistentFlags().Lookup("enable-crossaz-endpoint"))
_ = viper.BindPFlag("crossAZTimeout", rootCmd.PersistentFlags().Lookup("crossaz-timeout"))
_ = viper.BindPFlag("crossAZLabelSelector", rootCmd.PersistentFlags().Lookup("crossaz-label-selector"))
```

#### 3.3 Update InfrabinService Struct

**File**: `pkg/infrabin/infrabin.go`

Update the struct to include optional Kubernetes client:
```go
type InfrabinService struct {
    UnimplementedInfrabinServer
    STSClient                 aws.STSClient
    HealthService             HealthService
    intermittentErrorsCounter atomic.Int32
    K8sClient                 *k8s.Client // Optional: nil if crossaz endpoint disabled
}
```

#### 3.4 Implement CrossAZ Method

**File**: `pkg/infrabin/crossaz.go` (new file)

```go
package infrabin

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "sync"
    "time"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "github.com/maruina/go-infrabin/internal/helpers"
    "github.com/maruina/go-infrabin/internal/k8s"
    "github.com/spf13/viper"
)

// CrossAZ implements the cross-availability-zone connectivity testing endpoint.
// It discovers all go-infrabin pods in different AZs and tests connectivity to them.
func (s *InfrabinService) CrossAZ(ctx context.Context, _ *Empty) (*CrossAZResponse, error) {
    // Check if endpoint is enabled
    if !viper.GetBool("enableCrossAZEndpoint") {
        return nil, status.Error(codes.Unimplemented, "crossaz endpoint is disabled. Enable with --enable-crossaz-endpoint")
    }

    // Check if Kubernetes client is available
    if s.K8sClient == nil {
        return nil, status.Error(codes.Internal, "Kubernetes client not initialized")
    }

    // Get current pod's AZ from environment variable
    currentAZ := helpers.GetEnv("AVAILABILITY_ZONE", "")
    if currentAZ == "" {
        return nil, status.Error(codes.FailedPrecondition, "AVAILABILITY_ZONE environment variable not set")
    }

    // Get current pod name
    currentPodName := helpers.GetEnv("POD_NAME", "K8S_POD_NAME", "")
    if currentPodName == "" {
        return nil, status.Error(codes.FailedPrecondition, "POD_NAME environment variable not set")
    }

    // Discover pods using label selector
    // Note: DiscoverPods now handles AZ extraction from both pod and node
    labelSelector := viper.GetString("crossAZLabelSelector")
    pods, err := s.K8sClient.DiscoverPods(ctx, labelSelector)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to discover pods: %v", err)
    }

    // Group pods by AZ and filter out current pod
    discoveredPods := groupPodsByAZ(pods, currentPodName)

    // Get pods in different AZs for testing
    crossAZPods := filterCrossAZPods(pods, currentAZ, currentPodName)

    // Test connectivity to cross-AZ pods in parallel
    testResults := s.testCrossAZConnectivity(ctx, crossAZPods)

    // Calculate summary statistics
    summary := calculateSummary(pods, discoveredPods, testResults)

    return &CrossAZResponse{
        CurrentAz:      currentAZ,
        CurrentPod:     currentPodName,
        DiscoveredPods: discoveredPods,
        CrossAzTests:   testResults,
        Summary:        summary,
    }, nil
}

// groupPodsByAZ groups pods by their availability zone.
func groupPodsByAZ(pods []k8s.PodInfo, currentPodName string) map[string]*PodList {
    result := make(map[string]*PodList)

    for _, pod := range pods {
        if pod.AvailabilityZone == "" {
            pod.AvailabilityZone = "unknown"
        }

        if _, exists := result[pod.AvailabilityZone]; !exists {
            result[pod.AvailabilityZone] = &PodList{PodNames: []string{}}
        }

        result[pod.AvailabilityZone].PodNames = append(
            result[pod.AvailabilityZone].PodNames,
            pod.Name,
        )
    }

    return result
}

// filterCrossAZPods returns pods that are in a different AZ than the current pod.
func filterCrossAZPods(pods []k8s.PodInfo, currentAZ, currentPodName string) []k8s.PodInfo {
    var result []k8s.PodInfo

    for _, pod := range pods {
        // Skip current pod
        if pod.Name == currentPodName {
            continue
        }

        // Skip pods in same AZ
        if pod.AvailabilityZone == currentAZ {
            continue
        }

        result = append(result, pod)
    }

    return result
}

// testCrossAZConnectivity tests connectivity to pods in parallel.
func (s *InfrabinService) testCrossAZConnectivity(ctx context.Context, pods []k8s.PodInfo) []*CrossAZTest {
    if len(pods) == 0 {
        return []*CrossAZTest{}
    }

    results := make([]*CrossAZTest, len(pods))
    var wg sync.WaitGroup

    for i, pod := range pods {
        wg.Add(1)
        go func(index int, podInfo k8s.PodInfo) {
            defer wg.Done()
            results[index] = s.testPodConnectivity(ctx, podInfo)
        }(i, pod)
    }

    wg.Wait()
    return results
}

// testPodConnectivity tests connectivity to a single pod.
func (s *InfrabinService) testPodConnectivity(ctx context.Context, pod k8s.PodInfo) *CrossAZTest {
    timeout := viper.GetDuration("crossAZTimeout")
    testURL := fmt.Sprintf("http://%s:8888/", pod.IP)

    // Create HTTP client with timeout
    client := &http.Client{
        Timeout: timeout,
    }

    start := time.Now()
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
    if err != nil {
        return &CrossAZTest{
            PodName:    pod.Name,
            PodIp:      pod.IP,
            Az:         pod.AvailabilityZone,
            Success:    false,
            StatusCode: 0,
            DurationMs: time.Since(start).Milliseconds(),
            Error:      fmt.Sprintf("failed to create request: %v", err),
        }
    }

    resp, err := client.Do(req)
    duration := time.Since(start)

    if err != nil {
        return &CrossAZTest{
            PodName:    pod.Name,
            PodIp:      pod.IP,
            Az:         pod.AvailabilityZone,
            Success:    false,
            StatusCode: 0,
            DurationMs: duration.Milliseconds(),
            Error:      err.Error(),
        }
    }
    defer resp.Body.Close()

    // Consider 2xx status codes as success
    success := resp.StatusCode >= 200 && resp.StatusCode < 300
    errorMsg := ""
    if !success {
        errorMsg = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
    }

    return &CrossAZTest{
        PodName:    pod.Name,
        PodIp:      pod.IP,
        Az:         pod.AvailabilityZone,
        Success:    success,
        StatusCode: int32(resp.StatusCode),
        DurationMs: duration.Milliseconds(),
        Error:      errorMsg,
    }
}

// calculateSummary computes aggregate statistics for the response.
func calculateSummary(allPods []k8s.PodInfo, discoveredPods map[string]*PodList, testResults []*CrossAZTest) *CrossAZSummary {
    successful := int32(0)
    failed := int32(0)

    for _, result := range testResults {
        if result.Success {
            successful++
        } else {
            failed++
        }
    }

    return &CrossAZSummary{
        TotalPods:          int32(len(allPods)),
        TotalAzs:           int32(len(discoveredPods)),
        CrossAzPodsTested:  int32(len(testResults)),
        SuccessfulTests:    successful,
        FailedTests:        failed,
    }
}
```

**Key Design Decisions**:
- Check if endpoint is enabled before proceeding
- Validate required environment variables (AVAILABILITY_ZONE, POD_NAME)
- Use goroutines for parallel connectivity tests (configured answer: parallel)
- No caching of pod discovery results (configured answer: always fetch fresh)
- Consider 2xx status codes as success
- Graceful error handling with detailed error messages

#### 3.5 Add Prometheus Metrics

**File**: `pkg/infrabin/metrics.go`

Add new metrics:
```go
var (
    // ... existing metrics ...

    // CrossAZ metrics
    crossAZTestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "crossaz_tests_total",
            Help: "Total number of cross-AZ connectivity tests performed",
        },
        []string{"source_az", "target_az", "result"},
    )

    crossAZTestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "crossaz_test_duration_milliseconds",
            Help:    "Duration of cross-AZ connectivity tests in milliseconds",
            Buckets: prometheus.ExponentialBuckets(10, 2, 10), // 10ms to ~10s
        },
        []string{"source_az", "source_pod", "target_az", "destination_pod"},
    )

    crossAZPodsDiscovered = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "crossaz_pods_discovered",
            Help: "Number of pods discovered per availability zone",
        },
        []string{"az"},
    )
)
```

Update `testPodConnectivity` in `crossaz.go` to record metrics:
```go
// Record metrics
defer func() {
    result := "success"
    if !success {
        result = "failure"
    }

    sourceAZ := helpers.GetEnv("AVAILABILITY_ZONE", "unknown")
    crossAZTestsTotal.WithLabelValues(sourceAZ, pod.AvailabilityZone, result).Inc()
    crossAZTestDuration.WithLabelValues(sourceAZ, pod.AvailabilityZone).Observe(float64(duration.Milliseconds()))
}()
```

Update `groupPodsByAZ` to record discovery metrics:
```go
// Record discovery metrics
for az, podList := range result {
    crossAZPodsDiscovered.WithLabelValues(az).Set(float64(len(podList.PodNames)))
}
```

---

### Phase 4: Initialize Kubernetes Client

**File**: `internal/cmd/root.go` (or wherever service initialization happens)

Update service initialization to optionally create Kubernetes client:

```go
// Initialize Kubernetes client if crossaz endpoint is enabled
var k8sClient *k8s.Client
if viper.GetBool("enableCrossAZEndpoint") {
    client, err := k8s.NewInClusterClient()
    if err != nil {
        log.Printf("WARNING: Failed to initialize Kubernetes client: %v", err)
        log.Printf("CrossAZ endpoint will not be available")
    } else {
        k8sClient = client
        log.Printf("Kubernetes client initialized for CrossAZ endpoint")
    }
}

// Create infrabin service with optional K8s client
infrabinService := &infrabin.InfrabinService{
    STSClient:     aws.NewSTSClient(),
    HealthService: healthService,
    K8sClient:     k8sClient,
}
```

---

### Phase 5: Helm Chart Updates

#### 5.1 Update Values

**File**: `charts/go-infrabin/values.yaml`

Add new configuration section:
```yaml
args:
  # ... existing args ...
  enableCrossAZEndpoint: false
  crossAZTimeout: 3s
  crossAZLabelSelector: "app.kubernetes.io/name=go-infrabin"

# RBAC configuration for CrossAZ endpoint
rbac:
  # Specifies whether PSP resources should be created
  pspEnabled: false
  # Enable RBAC for CrossAZ endpoint (requires cluster admin)
  crossAZEnabled: false
```

#### 5.2 Create RBAC Resources

**File**: `charts/go-infrabin/templates/rbac.yaml` (new file)

```yaml
{{- if .Values.rbac.crossAZEnabled }}
---
# Role for namespace-scoped resources (pods)
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "go-infrabin.fullname" . }}-crossaz
  labels:
    {{- include "go-infrabin.labels" . | nindent 4 }}
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "go-infrabin.fullname" . }}-crossaz
  labels:
    {{- include "go-infrabin.labels" . | nindent 4 }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ include "go-infrabin.fullname" . }}-crossaz
subjects:
  - kind: ServiceAccount
    name: {{ include "go-infrabin.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
---
# ClusterRole for cluster-scoped resources (nodes)
# Note: Nodes are cluster-scoped, so we need ClusterRole to read them
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "go-infrabin.fullname" . }}-crossaz-nodes
  labels:
    {{- include "go-infrabin.labels" . | nindent 4 }}
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ include "go-infrabin.fullname" . }}-crossaz-nodes
  labels:
    {{- include "go-infrabin.labels" . | nindent 4 }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{ include "go-infrabin.fullname" . }}-crossaz-nodes
subjects:
  - kind: ServiceAccount
    name: {{ include "go-infrabin.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
{{- end }}
```

**Important Note**:
- Nodes are cluster-scoped resources, requiring `ClusterRole` for read access
- This grants the service account permission to read (but not modify) all nodes in the cluster
- This is necessary for the `extractAZ()` function to query node labels
- If you cannot grant cluster-level permissions, ensure all pods have the AZ in their node selector to avoid node queries

#### 5.3 Update Deployment

**File**: `charts/go-infrabin/templates/deployment.yaml`

Add CrossAZ flags to command (after existing flags):
```yaml
{{- if .Values.args.enableCrossAZEndpoint }}
- --enable-crossaz-endpoint=true
- --crossaz-timeout={{ .Values.args.crossAZTimeout }}
- --crossaz-label-selector={{ .Values.args.crossAZLabelSelector }}
{{- end }}
```

Add Downward API environment variables (in env section):

```yaml
env:
{{- if .Values.args.enableCrossAZEndpoint }}
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: NODE_NAME
  valueFrom:
    fieldRef:
      fieldPath: spec.nodeName
{{- end }}
```

Then in the code, we'll query the node to get its AZ label and cache it at startup.

Remember to add the label selector to the template helpers

---

### Phase 6: Testing

#### 6.1 Unit Tests

**File**: `pkg/infrabin/crossaz_test.go`

```go
package infrabin

import (
    "testing"

    "github.com/maruina/go-infrabin/internal/k8s"
)

func TestGroupPodsByAZ(t *testing.T) {
    pods := []k8s.PodInfo{
        {Name: "pod1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
        {Name: "pod2", IP: "10.0.1.2", AvailabilityZone: "us-east-1a"},
        {Name: "pod3", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
    }

    result := groupPodsByAZ(pods, "pod1")

    if len(result) != 2 {
        t.Errorf("Expected 2 AZs, got %d", len(result))
    }

    if len(result["us-east-1a"].PodNames) != 2 {
        t.Errorf("Expected 2 pods in us-east-1a, got %d", len(result["us-east-1a"].PodNames))
    }

    if len(result["us-east-1b"].PodNames) != 1 {
        t.Errorf("Expected 1 pod in us-east-1b, got %d", len(result["us-east-1b"].PodNames))
    }
}

func TestFilterCrossAZPods(t *testing.T) {
    pods := []k8s.PodInfo{
        {Name: "pod1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
        {Name: "pod2", IP: "10.0.1.2", AvailabilityZone: "us-east-1a"},
        {Name: "pod3", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
        {Name: "pod4", IP: "10.0.3.1", AvailabilityZone: "us-east-1c"},
    }

    result := filterCrossAZPods(pods, "us-east-1a", "pod1")

    // Should exclude pod1 (current pod) and pod2 (same AZ)
    if len(result) != 2 {
        t.Errorf("Expected 2 cross-AZ pods, got %d", len(result))
    }

    for _, pod := range result {
        if pod.AvailabilityZone == "us-east-1a" {
            t.Errorf("Found pod in same AZ: %s", pod.Name)
        }
        if pod.Name == "pod1" {
            t.Errorf("Found current pod in results: %s", pod.Name)
        }
    }
}

func TestCalculateSummary(t *testing.T) {
    allPods := []k8s.PodInfo{
        {Name: "pod1", IP: "10.0.1.1", AvailabilityZone: "us-east-1a"},
        {Name: "pod2", IP: "10.0.2.1", AvailabilityZone: "us-east-1b"},
        {Name: "pod3", IP: "10.0.3.1", AvailabilityZone: "us-east-1c"},
    }

    discoveredPods := map[string]*PodList{
        "us-east-1a": {PodNames: []string{"pod1"}},
        "us-east-1b": {PodNames: []string{"pod2"}},
        "us-east-1c": {PodNames: []string{"pod3"}},
    }

    testResults := []*CrossAZTest{
        {PodName: "pod2", Success: true},
        {PodName: "pod3", Success: false},
    }

    summary := calculateSummary(allPods, discoveredPods, testResults)

    if summary.TotalPods != 3 {
        t.Errorf("Expected 3 total pods, got %d", summary.TotalPods)
    }

    if summary.TotalAzs != 3 {
        t.Errorf("Expected 3 AZs, got %d", summary.TotalAzs)
    }

    if summary.CrossAzPodsTested != 2 {
        t.Errorf("Expected 2 tested pods, got %d", summary.CrossAzPodsTested)
    }

    if summary.SuccessfulTests != 1 {
        t.Errorf("Expected 1 successful test, got %d", summary.SuccessfulTests)
    }

    if summary.FailedTests != 1 {
        t.Errorf("Expected 1 failed test, got %d", summary.FailedTests)
    }
}
```

**File**: `internal/k8s/client_test.go`

```go
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
                t.Errorf("Unexpected error: %v", err)
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
    result, err := client.extractAZ(ctx, pod)
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
    if result != "unknown" {
        t.Errorf("Expected 'unknown', got %s", result)
    }
}
```

#### 6.2 Integration Test Plan

Create integration test that:
1. Deploys go-infrabin with crossaz enabled in test cluster
2. Uses `kubectl port-forward` to access the endpoint
3. Calls `/crossaz` endpoint
4. Validates response structure
5. Checks that pods are discovered
6. Verifies connectivity tests run

---

### Phase 7: Documentation

#### 7.1 Update README

**File**: `README.md`

Add to API Documentation table:
```markdown
| `GET /crossaz` | Cross-AZ connectivity test (requires `--enable-crossaz-endpoint`) |
```

Add new section after Health Check Endpoints:

```markdown
#### Cross-AZ Connectivity Endpoint

The `/crossaz` endpoint enables automatic discovery and connectivity testing across availability zones in Kubernetes. This is useful for identifying network misconfigurations that prevent cross-AZ communication.

**Prerequisites**:
- Running in Kubernetes with appropriate RBAC permissions
- Node labels with AZ information (`topology.kubernetes.io/zone`)
- Service account with pod list/get permissions

**Configuration**:
```bash
# Enable the endpoint (disabled by default)
--enable-crossaz-endpoint=true

# Configure timeout for connectivity tests (default: 3s)
--crossaz-timeout=5s

# Configure label selector for pod discovery (default: app.kubernetes.io/name=go-infrabin)
--crossaz-label-selector="app.kubernetes.io/name=go-infrabin"
```

**Example Usage**:
```bash
# Get cross-AZ connectivity status
curl http://localhost:8888/crossaz
```

**Response**:
```json
{
  "current_az": "us-east-1a",
  "current_pod": "go-infrabin-abc123",
  "discovered_pods": {
    "us-east-1a": ["go-infrabin-abc123", "go-infrabin-def456"],
    "us-east-1b": ["go-infrabin-ghi789"]
  },
  "cross_az_tests": [
    {
      "pod_name": "go-infrabin-ghi789",
      "pod_ip": "10.0.2.15",
      "az": "us-east-1b",
      "success": true,
      "status_code": 200,
      "duration_ms": 45
    }
  ],
  "summary": {
    "total_pods": 3,
    "total_azs": 2,
    "cross_az_pods_tested": 1,
    "successful_tests": 1,
    "failed_tests": 0
  }
}
```

**Prometheus Metrics**:
The endpoint exposes the following metrics on the `/metrics` endpoint:
- `crossaz_tests_total{source_az, target_az, result}` - Total number of tests
- `crossaz_test_duration_milliseconds{source_az, target_az}` - Test duration histogram
- `crossaz_pods_discovered{az}` - Number of pods per AZ
```

#### 7.2 Update Helm Chart README

**File**: `charts/go-infrabin/README.md`

Add configuration section for CrossAZ feature with RBAC setup instructions.

---

## Implementation Order

1. **Phase 1**: Add dependencies and proto definitions (foundational)
2. **Phase 2**: Implement Kubernetes client package (can be tested independently)
3. **Phase 3**: Implement CrossAZ logic (depends on Phase 2)
4. **Phase 4**: Wire up initialization (integrates Phase 2 & 3)
5. **Phase 5**: Update Helm chart (enables Kubernetes deployment)
6. **Phase 6**: Add tests (validates implementation)
7. **Phase 7**: Update documentation (completes feature)

## Testing Strategy

### Unit Testing
- Test each function in isolation with mock data
- Focus on: pod filtering, AZ grouping, summary calculation
- Use table-driven tests for multiple scenarios

### Integration Testing
- Deploy in real Kubernetes cluster with multiple AZs
- Verify RBAC permissions work correctly
- Test endpoint disabled/enabled behavior
- Simulate network failures with network policies

### Manual Testing
- Deploy 3+ replicas across different AZs
- Call endpoint and verify response
- Check Prometheus metrics are recorded
- Test error scenarios (missing env vars, no RBAC, etc.)

## Rollback Plan

If issues are discovered:
1. Disable endpoint via `--enable-crossaz-endpoint=false` (feature flag)
2. Remove RBAC resources if they cause security concerns
3. Revert proto changes requires regeneration
4. Remove Kubernetes dependencies if they cause conflicts

## Success Criteria

Implementation complete when:
- [ ] All code compiles and proto generates successfully
- [ ] Unit tests pass with >80% coverage
- [ ] Integration tests pass in test cluster
- [ ] Endpoint returns valid JSON responses
- [ ] RBAC permissions work correctly
- [ ] Prometheus metrics are recorded
- [ ] Documentation is updated
- [ ] Manual testing checklist is complete

## Decisions Made

1. **Unified AZ Extraction**: ✅ Implemented single `extractAZ()` function that:
   - First checks pod's node selector (fast, no API call)
   - Falls back to querying node labels via Node API (requires ClusterRole)
   - Returns "unknown" on failure (graceful degradation)

2. **RBAC Requirements**: ✅ Requires both:
   - Namespace-scoped Role for pods (`get`, `list`)
   - Cluster-scoped ClusterRole for nodes (`get`)

3. **Prometheus Metrics**: ✅ Enhanced with pod-level labels:
   - `crossaz_test_duration_milliseconds{source_az, source_pod, target_az, destination_pod}`
   - Allows tracking specific pod-to-pod connectivity patterns

## Open Questions

1. **Error Handling**: Should we fail the entire request if Kubernetes API is unavailable, or return partial results?
   - **Current plan**: Fail fast with 500 error if K8s client is nil
   - Individual pod discovery errors are logged but don't fail the entire request
   - AZ extraction errors result in "unknown" AZ but don't fail pod discovery

2. **Rate Limiting**: Should we implement rate limiting for the endpoint to prevent API server overload?
   - **Current plan**: No rate limiting, rely on Kubernetes API server built-in rate limiting
   - Node queries are made per-pod which could be expensive in large clusters
   - Consider adding optional caching of node AZ labels if this becomes an issue

3. **Metrics Cardinality**: With dynamic AZ and pod labels, metrics cardinality could be high. Should we limit label values?
   - **Current plan**: No limits, assume reasonable number of AZs and pods
   - In large clusters, may need to consider cardinality limits or aggregation
