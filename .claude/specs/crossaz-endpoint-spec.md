# Cross-AZ Connectivity Endpoint Specification

## Purpose

Add a new `/crossaz` endpoint to go-infrabin that enables automatic discovery and connectivity testing across availability zones (AZs) in Kubernetes environments. This endpoint will help identify cross-AZ network misconfigurations by automatically discovering peer pods in other AZs and testing connectivity to them.

## User Problem

When running distributed systems across multiple availability zones in Kubernetes, network misconfigurations can prevent pods in one AZ from communicating with pods in another AZ. Currently, there's no automated way in go-infrabin to:

1. Discover which AZ a pod is running in
2. Find other go-infrabin pods running in different AZs
3. Test connectivity between AZs
4. Identify cross-AZ network issues

Users must manually identify pod IPs, determine their AZ placement, and test connectivity - a tedious and error-prone process.

## Success Criteria

The `/crossaz` endpoint implementation is successful when:

1. A pod can automatically determine its own availability zone
2. The pod can discover all other go-infrabin pods in the same namespace
3. The pod can identify which pods are in different AZs
4. The pod can test connectivity to pods in other AZs
5. The response clearly indicates success/failure for each cross-AZ connection
6. The endpoint works with appropriate RBAC permissions in Kubernetes
7. The endpoint returns useful diagnostic information when failures occur
8. The feature can be optionally enabled/disabled via configuration

## Scope

### In Scope

1. **AZ Detection**
   - Use Kubernetes Downward API to inject AZ information via environment variable
   - Support standard topology labels: `topology.kubernetes.io/zone`
   - Fallback to legacy label: `failure-domain.beta.kubernetes.io/zone`

2. **Pod Discovery**
   - Use Kubernetes client-go library to query pod list
   - Filter pods by namespace (same namespace as caller)
   - Filter pods by label selector to identify go-infrabin pods
   - Extract pod IP addresses and AZ labels

3. **Connectivity Testing**
   - Make HTTP calls to discovered pods in different AZs
   - Test against the `/` (root) endpoint of peer pods
   - Capture timing information for each request
   - Handle timeouts and connection failures gracefully

4. **Response Format**
   - Return JSON response with:
     - Current pod's AZ
     - List of discovered pods per AZ
     - Connectivity test results for each cross-AZ pod
     - Success/failure status
     - Error messages for failures
     - Timing information

5. **Kubernetes Integration**
   - Add RBAC resources (Role/RoleBinding) to Helm chart
   - Grant `get`, `list` permissions for pods in namespace
   - Use existing ServiceAccount or create dedicated one
   - Configure Downward API in pod spec to inject AZ label

6. **Configuration**
   - Optional flag to enable/disable the endpoint (default: disabled for security)
   - Configurable timeout for cross-AZ calls
   - Configurable label selector for pod discovery

### Out of Scope

1. Cross-namespace pod discovery
2. Testing connectivity to pods in the same AZ (focus is cross-AZ only)
3. Advanced network diagnostics (traceroute, MTU discovery, etc.)
4. Persistent storage of test results
5. Metrics/observability integration (may be added in future)
6. Testing non-go-infrabin pods
7. Support for multi-cluster scenarios
8. AWS-specific metadata service integration for AZ detection

## Technical Considerations

### Architecture Decisions

1. **Kubernetes Client Initialization**
   - Use in-cluster configuration (automatic service account token)
   - Initialize client once at startup, reuse for all requests
   - Handle client initialization errors gracefully

2. **Pod Discovery Strategy**
   - Use label selector: `app.kubernetes.io/name=go-infrabin` (matches Helm chart labels)
   - Query pods in the same namespace only
   - Filter out the calling pod itself from results
   - Cache pod list for short duration (e.g., 5 seconds) to reduce API load

3. **AZ Detection Method**
   - Primary: Environment variable injected via Downward API
   - Read from `AVAILABILITY_ZONE` environment variable
   - Return error if AZ cannot be determined (fail-fast)

4. **Connectivity Test Implementation**
   - Reuse existing HTTP client code patterns
   - Make concurrent requests to pods in parallel (with goroutines)
   - Use configurable timeout (default: 3 seconds)
   - Test against `http://<pod-ip>:8888/` endpoint
   - Collect timing and status code information

5. **Error Handling**
   - Return 503 Service Unavailable if endpoint is disabled
   - Return 500 Internal Server Error for Kubernetes API failures
   - Return 200 OK with partial results if some pods fail (include error details per pod)
   - Include clear error messages for debugging

6. **Security Considerations**
   - Endpoint disabled by default (requires `--enable-crossaz-endpoint` flag)
   - RBAC limited to pod read access in namespace only
   - No write operations on Kubernetes API
   - Rate limiting considerations for API calls

### Dependencies

1. **New Go Dependencies**
   - `k8s.io/client-go` - Kubernetes Go client library
   - `k8s.io/api` - Kubernetes API types
   - `k8s.io/apimachinery` - Kubernetes API machinery

2. **Helm Chart Changes**
   - Add RBAC Role with pod list/get permissions
   - Add RoleBinding to ServiceAccount
   - Add Downward API volume/environment variable for AZ
   - Add new configuration values for endpoint enablement

3. **Configuration Changes**
   - Add `--enable-crossaz-endpoint` flag (default: false)
   - Add `--crossaz-timeout` flag (default: 3s)
   - Add `--crossaz-label-selector` flag (default: `app.kubernetes.io/name=go-infrabin`)

### API Design

**Endpoint**: `GET /crossaz`

**Request**: No parameters required

**Response** (200 OK):
```json
{
  "current_az": "us-east-1a",
  "current_pod": "go-infrabin-abc123",
  "discovered_pods": {
    "us-east-1a": ["go-infrabin-abc123", "go-infrabin-def456"],
    "us-east-1b": ["go-infrabin-ghi789"],
    "us-east-1c": ["go-infrabin-jkl012"]
  },
  "cross_az_tests": [
    {
      "pod_name": "go-infrabin-ghi789",
      "pod_ip": "10.0.2.15",
      "az": "us-east-1b",
      "success": true,
      "status_code": 200,
      "duration_ms": 45,
      "error": ""
    },
    {
      "pod_name": "go-infrabin-jkl012",
      "pod_ip": "10.0.3.20",
      "az": "us-east-1c",
      "success": false,
      "status_code": 0,
      "duration_ms": 3000,
      "error": "context deadline exceeded"
    }
  ],
  "summary": {
    "total_pods": 4,
    "total_azs": 3,
    "cross_az_pods_tested": 2,
    "successful_tests": 1,
    "failed_tests": 1
  }
}
```

**Response** (503 Service Unavailable):
```json
{
  "error": "crossaz endpoint is disabled. Enable with --enable-crossaz-endpoint"
}
```

**Response** (500 Internal Server Error):
```json
{
  "error": "failed to determine availability zone: AVAILABILITY_ZONE environment variable not set"
}
```

### Proto Definition

Add to `proto/infrabin/infrabin.proto`:

```protobuf
// CrossAZ performs cross-availability-zone connectivity tests.
// Discovers all go-infrabin pods in different AZs and tests connectivity to them.
// Requires --enable-crossaz-endpoint flag and appropriate RBAC permissions.
rpc CrossAZ(Empty) returns (CrossAZResponse) {
    option (google.api.http) = {
        get: "/crossaz"
    };
}

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

## Testing Requirements

### Unit Tests

1. Test AZ detection from environment variables
2. Test pod filtering logic (label selector, namespace, exclude self)
3. Test cross-AZ test result aggregation
4. Mock Kubernetes client for pod discovery tests
5. Test error handling for missing AZ environment variable
6. Test error handling for Kubernetes API failures

### Integration Tests

1. Deploy multiple go-infrabin pods across different AZs in test cluster
2. Verify `/crossaz` endpoint discovers all pods
3. Verify connectivity tests succeed between AZs
4. Test with network policies to simulate failures
5. Verify RBAC permissions work correctly
6. Test endpoint disabled/enabled behavior
7. Test with missing Downward API configuration

### Manual Testing Checklist

- [ ] Deploy go-infrabin in 3 AZs
- [ ] Verify each pod can see pods in other AZs
- [ ] Verify cross-AZ connectivity tests succeed
- [ ] Test with endpoint disabled (should return 503)
- [ ] Test with missing RBAC permissions (should fail gracefully)
- [ ] Test with missing AZ environment variable (should return 500)
- [ ] Verify response JSON structure matches spec
- [ ] Test gRPC endpoint as well as REST endpoint
- [ ] Verify timing information is accurate

## Constraints

1. **Kubernetes Version**: Requires Kubernetes 1.19+ (for `topology.kubernetes.io/zone` label)
2. **RBAC**: Requires cluster admin to install with RBAC permissions
3. **Network**: Assumes pod-to-pod networking is functional in same AZ
4. **Label Convention**: Assumes nodes are labeled with AZ information
5. **Performance**: Pod discovery API call may be expensive in large clusters (consider caching)

## Alternative Approaches Considered

### Alternative 1: Use AWS Metadata Service for AZ Detection
**Pros**: Works without Kubernetes labels/Downward API
**Cons**: AWS-specific, doesn't work in other clouds or on-premises
**Decision**: Rejected. Keep implementation cloud-agnostic using Kubernetes primitives.

### Alternative 2: Use Headless Service for Pod Discovery
**Pros**: No RBAC permissions required, DNS-based discovery
**Cons**: Can't filter by AZ, requires additional service configuration, DNS latency
**Decision**: Rejected. Kubernetes API provides more control and metadata.

### Alternative 3: Use DaemonSet Instead of Deployment
**Pros**: Guaranteed one pod per node, clearer AZ distribution
**Cons**: Changes deployment model, may not fit all use cases
**Decision**: Rejected. Keep flexible deployment options, let users choose.

### Alternative 4: External Service for Pod Discovery
**Pros**: No Kubernetes API access needed from pods
**Cons**: Additional complexity, external dependency, defeats purpose
**Decision**: Rejected. Direct Kubernetes API access is simpler and more reliable.

## Questions for User

Before proceeding to implementation:

1. **Endpoint Security**: Should the `/crossaz` endpoint require authentication/authorization, or is the enable flag sufficient? The enable flag is sufficient

2. **Label Selector**: Is `app.kubernetes.io/name=go-infrabin` the correct label selector, or should it be configurable/different? Check the current helm chart in @charts/go-infrabin and add any missin label

3. **Namespace Scope**: Should we support cross-namespace discovery with a flag, or keep it namespace-scoped for security? No, only the same namespace

4. **Port Configuration**: Should we test against port 8888, or should the port be configurable? 8888 is ok

5. **Caching Strategy**: Should we implement caching for pod discovery (e.g., 30s TTL), or always fetch fresh data? Always fetch fresh data

6. **Health Check Integration**: Should failed cross-AZ tests affect the liveness/readiness probes? No

7. **Metrics**: Should we expose Prometheus metrics for cross-AZ test results (e.g., success rate, latency)? Yes

8. **Parallel vs Sequential**: Should connectivity tests run in parallel (faster) or sequentially (easier to debug)? Parallel

## Next Steps

After spec approval:

1. Create implementation plan with detailed code structure
2. Update proto definitions
3. Implement Kubernetes client initialization
4. Implement pod discovery logic
5. Implement connectivity testing
6. Update Helm chart with RBAC and Downward API
7. Write unit tests
8. Write integration tests
9. Update documentation
10. Test in real Kubernetes cluster
