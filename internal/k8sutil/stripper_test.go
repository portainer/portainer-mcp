package k8sutil

import (
	"bytes"
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
