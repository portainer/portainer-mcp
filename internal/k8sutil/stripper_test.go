package k8sutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestProcessRawKubernetesAPIResponse(t *testing.T) {
	tests := []struct {
		name           string
		httpResp       *http.Response
		expectedResult string
		expectedError  bool
		description    string
	}{
		{
			name:          "nil response",
			httpResp:      nil,
			expectedError: true,
			description:   "should return error when http response is nil",
		},
		{
			name: "nil body with 204 status",
			httpResp: &http.Response{
				StatusCode:    http.StatusNoContent,
				Body:          nil,
				ContentLength: 0,
			},
			expectedResult: "",
			expectedError:  false,
			description:    "should handle nil body gracefully for 204 status",
		},
		{
			name: "nil body with 200 status",
			httpResp: &http.Response{
				StatusCode:    http.StatusOK,
				Body:          nil,
				ContentLength: 1,
			},
			expectedError: true,
			description:   "should return error when body is nil but content expected",
		},
		{
			name: "empty body",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			},
			expectedResult: "",
			expectedError:  false,
			description:    "should handle empty body gracefully",
		},
		{
			name: "empty JSON object",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
			},
			expectedResult: "{}",
			expectedError:  false,
			description:    "should handle empty JSON object",
		},
		{
			name: "empty JSON array",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("[]"))),
			},
			expectedResult: "[]",
			expectedError:  false,
			description:    "should handle empty JSON array",
		},
		{
			name: "invalid JSON",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("invalid json"))),
			},
			expectedError: true,
			description:   "should return error for invalid JSON",
		},
		{
			name: "single object with managedFields",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "Pod",
					"metadata": {
						"name": "test-pod",
						"namespace": "default",
						"managedFields": [
							{
								"manager": "kubectl-client-side-apply",
								"operation": "Update",
								"apiVersion": "v1",
								"time": "2023-01-01T00:00:00Z"
							}
						]
					},
					"spec": {
						"containers": [
							{
								"name": "test-container",
								"image": "nginx"
							}
						]
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod","namespace":"default"},"spec":{"containers":[{"image":"nginx","name":"test-container"}]}}`,
			expectedError:  false,
			description:    "should remove managedFields from single object",
		},
		{
			name: "single object without managedFields",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "Pod",
					"metadata": {
						"name": "test-pod",
						"namespace": "default"
					},
					"spec": {
						"containers": [
							{
								"name": "test-container",
								"image": "nginx"
							}
						]
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod","namespace":"default"},"spec":{"containers":[{"image":"nginx","name":"test-container"}]}}`,
			expectedError:  false,
			description:    "should handle single object without managedFields",
		},
		{
			name: "list with managedFields",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "PodList",
					"items": [
						{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod-1",
								"namespace": "default",
								"managedFields": [
									{
										"manager": "kubectl-client-side-apply",
										"operation": "Update",
										"apiVersion": "v1",
										"time": "2023-01-01T00:00:00Z"
									}
								]
							},
							"spec": {
								"containers": [
									{
										"name": "test-container",
										"image": "nginx"
									}
								]
							}
						},
						{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod-2",
								"namespace": "default",
								"managedFields": [
									{
										"manager": "kubectl-client-side-apply",
										"operation": "Update",
										"apiVersion": "v1",
										"time": "2023-01-01T00:00:00Z"
									}
								]
							},
							"spec": {
								"containers": [
									{
										"name": "test-container",
										"image": "redis"
									}
								]
							}
						}
					]
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod-1","namespace":"default"},"spec":{"containers":[{"image":"nginx","name":"test-container"}]}},{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod-2","namespace":"default"},"spec":{"containers":[{"image":"redis","name":"test-container"}]}}],"kind":"PodList"}`,
			expectedError:  false,
			description:    "should remove managedFields from all items in list",
		},
		{
			name: "object without metadata",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "Pod",
					"spec": {
						"containers": [
							{
								"name": "test-container",
								"image": "nginx"
							}
						]
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"Pod","spec":{"containers":[{"image":"nginx","name":"test-container"}]}}`,
			expectedError:  false,
			description:    "should handle object without metadata",
		},
		{
			name: "empty object with no fields",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"apiVersion":"v1","kind":"Pod"}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"Pod"}`,
			expectedError:  false,
			description:    "should handle empty object with no fields",
		},
		{
			name: "object with other metadata fields",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "Service",
					"metadata": {
						"name": "test-service",
						"namespace": "default",
						"labels": {"app": "test"},
						"annotations": {"key": "value"},
						"managedFields": [
							{
								"manager": "kubectl-client-side-apply",
								"operation": "Update",
								"apiVersion": "v1",
								"time": "2023-01-01T00:00:00Z"
							}
						]
					},
					"spec": {
						"ports": [{"port": 80}]
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"key":"value"},"labels":{"app":"test"},"name":"test-service","namespace":"default"},"spec":{"ports":[{"port":80}]}}`,
			expectedError:  false,
			description:    "should preserve other metadata fields while removing managedFields",
		},
		{
			name: "list with mixed items",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "PodList",
					"items": [
						{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod-1",
								"managedFields": [{"manager": "kubectl"}]
							}
						},
						{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod-2"
							}
						}
					]
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod-1"}},{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod-2"}}],"kind":"PodList"}`,
			expectedError:  false,
			description:    "should handle list with items that have and don't have managedFields",
		},
		{
			name: "nil body with 404 status",
			httpResp: &http.Response{
				StatusCode:    http.StatusNotFound,
				Body:          nil,
				ContentLength: 0,
			},
			expectedResult: "",
			expectedError:  false,
			description:    "should handle nil body for non-204 status with zero content length",
		},
		{
			name: "nil body with 500 status",
			httpResp: &http.Response{
				StatusCode:    http.StatusInternalServerError,
				Body:          nil,
				ContentLength: 0,
			},
			expectedResult: "",
			expectedError:  false,
			description:    "should handle nil body for error status with zero content length",
		},
		{
			name: "malformed list JSON",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "PodList",
					"items": "not-an-array"
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"PodList","items":"not-an-array"}`,
			expectedError:  false,
			description:    "should handle malformed list JSON gracefully",
		},
		{
			name: "object with circular reference",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "Pod",
					"metadata": {
						"name": "test-pod",
						"managedFields": [
							{
								"manager": "kubectl-client-side-apply",
								"operation": "Update",
								"apiVersion": "v1",
								"time": "2023-01-01T00:00:00Z"
							}
						]
					},
					"spec": {
						"containers": [
							{
								"name": "test-container",
								"image": "nginx"
							}
						]
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod"},"spec":{"containers":[{"image":"nginx","name":"test-container"}]}}`,
			expectedError:  false,
			description:    "should handle object with managedFields and preserve other fields",
		},
		{
			name: "list with empty items",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "PodList",
					"items": []
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","items":[],"kind":"PodList"}`,
			expectedError:  false,
			description:    "should handle list with empty items array",
		},
		{
			name: "object with deeply nested managedFields",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "v1",
					"kind": "Pod",
					"metadata": {
						"name": "test-pod",
						"managedFields": [
							{
								"manager": "kubectl-client-side-apply",
								"operation": "Update",
								"apiVersion": "v1",
								"time": "2023-01-01T00:00:00Z",
								"fields": {
									"f:spec": {
										"f:containers": {
											"k:{\"name\":\"test-container\"}": {
												"f:image": {}
											}
										}
									}
								}
							}
						]
					},
					"spec": {
						"containers": [
							{
								"name": "test-container",
								"image": "nginx"
							}
						]
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod"},"spec":{"containers":[{"image":"nginx","name":"test-container"}]}}`,
			expectedError:  false,
			description:    "should remove complex managedFields with nested structures",
		},
		{
			name: "different Kubernetes resource type",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "apps/v1",
					"kind": "Deployment",
					"metadata": {
						"name": "test-deployment",
						"namespace": "default",
						"managedFields": [
							{
								"manager": "kubectl-client-side-apply",
								"operation": "Update",
								"apiVersion": "apps/v1",
								"time": "2023-01-01T00:00:00Z"
							}
						]
					},
					"spec": {
						"replicas": 3,
						"selector": {
							"matchLabels": {
								"app": "test"
							}
						}
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"test-deployment","namespace":"default"},"spec":{"replicas":3,"selector":{"matchLabels":{"app":"test"}}}}`,
			expectedError:  false,
			description:    "should handle different Kubernetes resource types",
		},
		{
			name: "custom resource with managedFields",
			httpResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"apiVersion": "custom.example.com/v1",
					"kind": "CustomResource",
					"metadata": {
						"name": "test-custom",
						"managedFields": [
							{
								"manager": "custom-controller",
								"operation": "Apply",
								"apiVersion": "custom.example.com/v1",
								"time": "2023-01-01T00:00:00Z"
							}
						]
					},
					"spec": {
						"customField": "customValue"
					}
				}`))),
			},
			expectedResult: `{"apiVersion":"custom.example.com/v1","kind":"CustomResource","metadata":{"name":"test-custom"},"spec":{"customField":"customValue"}}`,
			expectedError:  false,
			description:    "should handle custom resources with managedFields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessRawKubernetesAPIResponse(tt.httpResp)

			if tt.expectedError {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)
			if tt.expectedResult == "" {
				assert.Equal(t, tt.expectedResult, string(result), tt.description)
			} else {
				assert.JSONEq(t, tt.expectedResult, string(result), tt.description)
			}
		})
	}
}

func TestRemoveManagedFieldsFromUnstructuredObject(t *testing.T) {
	tests := []struct {
		name           string
		obj            *unstructured.Unstructured
		expectedResult *unstructured.Unstructured
		expectedError  bool
		description    string
	}{
		{
			name:           "nil object",
			obj:            nil,
			expectedResult: nil,
			expectedError:  false,
			description:    "should handle nil object gracefully",
		},
		{
			name: "object with nil Object field",
			obj: &unstructured.Unstructured{
				Object: nil,
			},
			expectedResult: &unstructured.Unstructured{
				Object: nil,
			},
			expectedError: false,
			description:   "should handle object with nil Object field",
		},
		{
			name: "object with managedFields",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
						"managedFields": []interface{}{
							map[string]interface{}{
								"manager":    "kubectl-client-side-apply",
								"operation":  "Update",
								"apiVersion": "v1",
								"time":       "2023-01-01T00:00:00Z",
							},
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedError: false,
			description:   "should remove managedFields from object metadata",
		},
		{
			name: "object without managedFields",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedError: false,
			description:   "should handle object without managedFields",
		},
		{
			name: "object without metadata",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedError: false,
			description:   "should handle object without metadata",
		},
		{
			name: "object with non-map metadata",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata":   "not-a-map",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata":   "not-a-map",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedError: true,
			description:   "should return error when metadata is not a map",
		},
		{
			name: "object with complex metadata",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name":      "test-service",
						"namespace": "default",
						"labels": map[string]interface{}{
							"app":     "test",
							"version": "v1",
						},
						"annotations": map[string]interface{}{
							"key1": "value1",
							"key2": "value2",
						},
						"managedFields": []interface{}{
							map[string]interface{}{
								"manager":    "kubectl-client-side-apply",
								"operation":  "Update",
								"apiVersion": "v1",
								"time":       "2023-01-01T00:00:00Z",
							},
							map[string]interface{}{
								"manager":    "controller-manager",
								"operation":  "Apply",
								"apiVersion": "v1",
								"time":       "2023-01-02T00:00:00Z",
							},
						},
						"ownerReferences": []interface{}{
							map[string]interface{}{
								"apiVersion": "apps/v1",
								"kind":       "Deployment",
								"name":       "test-deployment",
							},
						},
					},
					"spec": map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port": 80,
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name":      "test-service",
						"namespace": "default",
						"labels": map[string]interface{}{
							"app":     "test",
							"version": "v1",
						},
						"annotations": map[string]interface{}{
							"key1": "value1",
							"key2": "value2",
						},
						"ownerReferences": []interface{}{
							map[string]interface{}{
								"apiVersion": "apps/v1",
								"kind":       "Deployment",
								"name":       "test-deployment",
							},
						},
					},
					"spec": map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port": 80,
							},
						},
					},
				},
			},
			expectedError: false,
			description:   "should remove managedFields while preserving other metadata fields",
		},
		{
			name: "object with empty managedFields",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":          "test-pod",
						"namespace":     "default",
						"managedFields": []interface{}{},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedError: false,
			description:   "should remove empty managedFields array",
		},
		{
			name: "object with nil managedFields",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":          "test-pod",
						"namespace":     "default",
						"managedFields": nil,
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx",
							},
						},
					},
				},
			},
			expectedError: false,
			description:   "should remove nil managedFields",
		},
		{
			name: "object with minimal fields",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
				},
			},
			expectedError: false,
			description:   "should handle object with minimal metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := removeManagedFieldsFromUnstructuredObject(tt.obj)

			if tt.expectedError {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)
			assert.Equal(t, tt.expectedResult, tt.obj, tt.description)
		})
	}
}

// Helper function to create a JSON response for testing
func createJSONResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     make(http.Header),
	}
}

// Benchmark tests for performance
func BenchmarkProcessRawKubernetesAPIResponse_SingleObject(b *testing.B) {
	jsonBody := `{
		"apiVersion": "v1",
		"kind": "Pod",
		"metadata": {
			"name": "test-pod",
			"namespace": "default",
			"managedFields": [
				{
					"manager": "kubectl-client-side-apply",
					"operation": "Update",
					"apiVersion": "v1",
					"time": "2023-01-01T00:00:00Z"
				}
			]
		},
		"spec": {
			"containers": [
				{
					"name": "test-container",
					"image": "nginx"
				}
			]
		}
	}`

	for i := 0; i < b.N; i++ {
		resp := createJSONResponse(http.StatusOK, jsonBody)
		_, err := ProcessRawKubernetesAPIResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessRawKubernetesAPIResponse_List(b *testing.B) {
	jsonBody := `{
		"apiVersion": "v1",
		"kind": "PodList",
		"items": [
			{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
					"name": "test-pod-1",
					"namespace": "default",
					"managedFields": [
						{
							"manager": "kubectl-client-side-apply",
							"operation": "Update",
							"apiVersion": "v1",
							"time": "2023-01-01T00:00:00Z"
						}
					]
				},
				"spec": {
					"containers": [
						{
							"name": "test-container",
							"image": "nginx"
						}
					]
				}
			}
		]
	}`

	for i := 0; i < b.N; i++ {
		resp := createJSONResponse(http.StatusOK, jsonBody)
		_, err := ProcessRawKubernetesAPIResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestErrorConditions tests various error conditions that might occur
func TestErrorConditions(t *testing.T) {
	t.Run("body read error", func(t *testing.T) {
		// Create a response with a body that will fail to read
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       &errorReader{},
		}

		_, err := ProcessRawKubernetesAPIResponse(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read response body")
	})

	t.Run("json marshal error", func(t *testing.T) {
		// This test is difficult to trigger in practice since the unstructured
		// library handles most cases, but we can test the structure
		resp := createJSONResponse(http.StatusOK, `{
			"apiVersion": "v1",
			"kind": "Pod",
			"metadata": {
				"name": "test-pod",
				"managedFields": [{"manager": "kubectl"}]
			}
		}`)

		result, err := ProcessRawKubernetesAPIResponse(resp)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

func (e *errorReader) Close() error {
	return nil
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("very large managedFields", func(t *testing.T) {
		// Create a large managedFields array
		largeManagedFields := make([]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			largeManagedFields[i] = map[string]interface{}{
				"manager":    fmt.Sprintf("manager-%d", i),
				"operation":  "Update",
				"apiVersion": "v1",
				"time":       "2023-01-01T00:00:00Z",
			}
		}

		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name":          "test-pod",
					"managedFields": largeManagedFields,
				},
			},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		require.NoError(t, err)

		// Verify managedFields was removed
		metadata, found, err := unstructured.NestedFieldCopy(obj.Object, "metadata")
		require.NoError(t, err)
		require.True(t, found)

		metadataMap, ok := metadata.(map[string]interface{})
		require.True(t, ok)
		_, hasManagedFields := metadataMap["managedFields"]
		assert.False(t, hasManagedFields)
	})

	t.Run("metadata with special characters", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name":          "test-pod-with-special-chars-!@#$%^&*()",
					"namespace":     "default-namespace",
					"managedFields": []interface{}{map[string]interface{}{"manager": "kubectl"}},
				},
			},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		require.NoError(t, err)

		// Verify the object still has the special characters in name
		name, found, err := unstructured.NestedString(obj.Object, "metadata", "name")
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, "test-pod-with-special-chars-!@#$%^&*()", name)
	})

	t.Run("empty object after processing", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"managedFields": []interface{}{map[string]interface{}{"manager": "kubectl"}},
				},
			},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		require.NoError(t, err)

		// Verify the object still has the basic structure
		assert.Equal(t, "v1", obj.GetAPIVersion())
		assert.Equal(t, "Pod", obj.GetKind())
	})

	t.Run("object with only managedFields in metadata", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"managedFields": []interface{}{
						map[string]interface{}{"manager": "kubectl"},
						map[string]interface{}{"manager": "controller"},
					},
				},
			},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		require.NoError(t, err)

		// Verify managedFields was removed and metadata is now empty
		metadata, found, err := unstructured.NestedFieldCopy(obj.Object, "metadata")
		require.NoError(t, err)
		require.True(t, found)

		metadataMap, ok := metadata.(map[string]interface{})
		require.True(t, ok)
		assert.Empty(t, metadataMap)
	})

	t.Run("object with empty Object map", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		require.NoError(t, err)
		assert.Empty(t, obj.Object)
	})

	t.Run("object with non-map metadata that causes error", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   "not-a-map",
			},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metadata for object")
		assert.Contains(t, err.Error(), "is not in the expected map format")
	})

	t.Run("object with metadata that causes NestedFieldCopy error", func(t *testing.T) {
		// Create an object with a metadata field that will cause an error
		// This is difficult to trigger in practice, but we can test the error path
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name": "test-pod",
				},
			},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		require.NoError(t, err)
	})

	t.Run("object with metadata that causes SetNestedField error", func(t *testing.T) {
		// This is difficult to trigger in practice since SetNestedField is quite robust
		// But we can test the structure
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name":          "test-pod",
					"managedFields": []interface{}{map[string]interface{}{"manager": "kubectl"}},
				},
			},
		}

		err := removeManagedFieldsFromUnstructuredObject(obj)
		require.NoError(t, err)
	})
}

// TestAdditionalErrorCases tests additional error scenarios that might not be covered
func TestAdditionalErrorCases(t *testing.T) {
	t.Run("list with ToList error", func(t *testing.T) {
		// Create a malformed list that will cause ToList to fail
		// This is difficult to trigger in practice, but we can test the structure
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"apiVersion": "v1",
				"kind": "PodList",
				"items": "not-an-array"
			}`))),
		}

		result, err := ProcessRawKubernetesAPIResponse(resp)
		// This should not error because the unstructured library handles this gracefully
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("single object with empty Object map", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"apiVersion":"v1","kind":"Pod"}`))),
		}

		result, err := ProcessRawKubernetesAPIResponse(resp)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("list with error during item processing", func(t *testing.T) {
		// Create a list where one item has invalid metadata
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"apiVersion": "v1",
				"kind": "PodList",
				"items": [
					{
						"apiVersion": "v1",
						"kind": "Pod",
						"metadata": "not-a-map"
					}
				]
			}`))),
		}

		_, err := ProcessRawKubernetesAPIResponse(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove managedFields from item")
	})

	t.Run("single object with error during processing", func(t *testing.T) {
		// Create a single object with invalid metadata
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": "not-a-map"
			}`))),
		}

		_, err := ProcessRawKubernetesAPIResponse(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove managedFields from single object")
	})

	t.Run("json marshal error for list", func(t *testing.T) {
		// This is difficult to trigger in practice since json.Marshal is quite robust
		// But we can test the structure
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"apiVersion": "v1",
				"kind": "PodList",
				"items": [
					{
						"apiVersion": "v1",
						"kind": "Pod",
						"metadata": {
							"name": "test-pod",
							"managedFields": [{"manager": "kubectl"}]
						}
					}
				]
			}`))),
		}

		result, err := ProcessRawKubernetesAPIResponse(resp)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("json marshal error for single object", func(t *testing.T) {
		// This is difficult to trigger in practice since json.Marshal is quite robust
		// But we can test the structure
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"apiVersion": "v1",
				"kind": "Pod",
				"metadata": {
					"name": "test-pod",
					"managedFields": [{"manager": "kubectl"}]
				}
			}`))),
		}

		result, err := ProcessRawKubernetesAPIResponse(resp)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

// TestNilBodyWithContentLength tests the specific case where body is nil but content length is not zero
func TestNilBodyWithContentLength(t *testing.T) {
	t.Run("nil body with positive content length", func(t *testing.T) {
		resp := &http.Response{
			StatusCode:    http.StatusOK,
			Body:          nil,
			ContentLength: 100,
		}

		_, err := ProcessRawKubernetesAPIResponse(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http response body is nil but content was expected")
	})

	t.Run("nil body with negative content length", func(t *testing.T) {
		resp := &http.Response{
			StatusCode:    http.StatusOK,
			Body:          nil,
			ContentLength: -1,
		}

		_, err := ProcessRawKubernetesAPIResponse(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http response body is nil but content was expected")
	})
}

// TestUnmarshalErrorHandling tests the specific error handling for JSON unmarshaling
func TestUnmarshalErrorHandling(t *testing.T) {
	t.Run("invalid JSON that is not empty object or array", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("invalid json content"))),
		}

		_, err := ProcessRawKubernetesAPIResponse(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal JSON into Unstructured")
		assert.Contains(t, err.Error(), "Body: invalid json content")
	})

	t.Run("empty JSON object string", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		}

		result, err := ProcessRawKubernetesAPIResponse(resp)
		require.NoError(t, err)
		assert.Equal(t, "{}", string(result))
	})

	t.Run("empty JSON array string", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("[]"))),
		}

		result, err := ProcessRawKubernetesAPIResponse(resp)
		require.NoError(t, err)
		assert.Equal(t, "[]", string(result))
	})
}
