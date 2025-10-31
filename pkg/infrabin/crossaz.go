package infrabin

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/maruina/go-infrabin/internal/helpers"
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
	testResults := s.testCrossAZConnectivity(ctx, crossAZPods, currentAZ, currentPodName)

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
func groupPodsByAZ(pods []K8sPodInfo, currentPodName string) map[string]*PodList {
	result := make(map[string]*PodList)

	for _, pod := range pods {
		az := pod.AvailabilityZone
		if az == "" {
			az = "unknown"
		}

		if _, exists := result[az]; !exists {
			result[az] = &PodList{PodNames: []string{}}
		}

		result[az].PodNames = append(
			result[az].PodNames,
			pod.Name,
		)
	}

	// Record discovery metrics
	for az, podList := range result {
		crossAZPodsDiscovered.WithLabelValues(az).Set(float64(len(podList.PodNames)))
	}

	return result
}

// filterCrossAZPods returns pods that are in a different AZ than the current pod.
func filterCrossAZPods(pods []K8sPodInfo, currentAZ, currentPodName string) []K8sPodInfo {
	var result []K8sPodInfo

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
func (s *InfrabinService) testCrossAZConnectivity(ctx context.Context, pods []K8sPodInfo, sourceAZ, sourcePod string) []*CrossAZTest {
	if len(pods) == 0 {
		return []*CrossAZTest{}
	}

	results := make([]*CrossAZTest, len(pods))
	var wg sync.WaitGroup

	for i, pod := range pods {
		wg.Add(1)
		go func(index int, podInfo K8sPodInfo) {
			defer wg.Done()
			results[index] = s.testPodConnectivity(ctx, podInfo, sourceAZ, sourcePod)
		}(i, pod)
	}

	wg.Wait()
	return results
}

// testPodConnectivity tests connectivity to a single pod.
func (s *InfrabinService) testPodConnectivity(ctx context.Context, pod K8sPodInfo, sourceAZ, sourcePod string) *CrossAZTest {
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

	// Consider 2xx status codes as success
	success := false
	statusCode := int32(0)
	errorMsg := ""

	if err != nil {
		errorMsg = err.Error()
	} else {
		defer resp.Body.Close()
		statusCode = int32(resp.StatusCode)
		success = resp.StatusCode >= 200 && resp.StatusCode < 300
		if !success {
			errorMsg = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		}
	}

	// Record metrics
	result := "success"
	if !success {
		result = "failure"
	}
	crossAZTestsTotal.WithLabelValues(sourceAZ, pod.AvailabilityZone, result).Inc()
	crossAZTestDuration.WithLabelValues(sourceAZ, sourcePod, pod.AvailabilityZone, pod.Name).Observe(float64(duration.Milliseconds()))

	return &CrossAZTest{
		PodName:    pod.Name,
		PodIp:      pod.IP,
		Az:         pod.AvailabilityZone,
		Success:    success,
		StatusCode: statusCode,
		DurationMs: duration.Milliseconds(),
		Error:      errorMsg,
	}
}

// calculateSummary computes aggregate statistics for the response.
func calculateSummary(allPods []K8sPodInfo, discoveredPods map[string]*PodList, testResults []*CrossAZTest) *CrossAZSummary {
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
		TotalPods:         int32(len(allPods)),
		TotalAzs:          int32(len(discoveredPods)),
		CrossAzPodsTested: int32(len(testResults)),
		SuccessfulTests:   successful,
		FailedTests:       failed,
	}
}
