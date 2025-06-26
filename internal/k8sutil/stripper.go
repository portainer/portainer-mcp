package k8sutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// removeManagedFieldsFromUnstructuredObject is a helper function that modifies an Unstructured object in place
// by removing the managedFields attribute from its metadata.
func removeManagedFieldsFromUnstructuredObject(obj *unstructured.Unstructured) error {
	if obj == nil || obj.Object == nil {
		return nil // Nothing to do
	}

	metadata, found, err := unstructured.NestedFieldCopy(obj.Object, "metadata")
	if err != nil {
		return fmt.Errorf("error fetching metadata for object %s (%s): %w", obj.GetName(), obj.GetKind(), err)
	}
	if !found {
		return nil // Metadata not found, nothing to do
	}

	metadataMap, ok := metadata.(map[string]any)
	if !ok {
		return fmt.Errorf("metadata for object %s (%s) is not in the expected map format", obj.GetName(), obj.GetKind())
	}

	// Delete the managedFields key from the metadata map
	delete(metadataMap, "managedFields")

	// TODO: Consider also removing other verbose fields here, e.g., ownerReferences, if needed.
	// delete(metadataMap, "ownerReferences")

	// Set the modified metadata back to the object
	err = unstructured.SetNestedField(obj.Object, metadataMap, "metadata")
	if err != nil {
		return fmt.Errorf("error setting modified metadata for object %s (%s): %w", obj.GetName(), obj.GetKind(), err)
	}
	return nil
}

// ProcessRawKubernetesAPIResponse takes an HTTP response, processes the JSON body,
// removes managedFields (and potentially other verbose metadata) from any Kubernetes resource(s) found,
// and returns the modified JSON bytes.
func ProcessRawKubernetesAPIResponse(httpResp *http.Response) ([]byte, error) {
	if httpResp == nil {
		return nil, fmt.Errorf("http response is nil")
	}
	if httpResp.Body == nil {
		if httpResp.StatusCode != http.StatusNoContent && httpResp.ContentLength != 0 {
			return nil, fmt.Errorf("http response body is nil but content was expected (status: %s)", httpResp.Status)
		}
		return []byte{}, nil // Return empty bytes if no body and appropriate status
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(bodyBytes) == 0 {
		return bodyBytes, nil // Valid empty body
	}

	uObj := &unstructured.Unstructured{}
	if err := uObj.UnmarshalJSON(bodyBytes); err != nil {
		trimmedBody := string(bodyBytes)
		if trimmedBody == "{}" || trimmedBody == "[]" {
			return bodyBytes, nil // Valid empty JSON object/array
		}
		return nil, fmt.Errorf("failed to unmarshal JSON into Unstructured: %w. Body: %s", err, string(bodyBytes))
	}

	if uObj.IsList() {
		list, err := uObj.ToList()
		if err != nil {
			return nil, fmt.Errorf("failed to convert Unstructured to UnstructuredList: %w", err)
		}

		for i := range list.Items {
			if err := removeManagedFieldsFromUnstructuredObject(&list.Items[i]); err != nil {
				// Potentially log this error but continue processing other items
				// For now, we'll return on first error to be safe.
				return nil, fmt.Errorf("failed to remove managedFields from item %d in list: %w", i, err)
			}
		}
		return json.Marshal(list)
	} else {
		if len(uObj.Object) == 0 {
			return bodyBytes, nil // Empty object, nothing to process
		}
		if err := removeManagedFieldsFromUnstructuredObject(uObj); err != nil {
			return nil, fmt.Errorf("failed to remove managedFields from single object: %w", err)
		}
		return json.Marshal(uObj)
	}
}
