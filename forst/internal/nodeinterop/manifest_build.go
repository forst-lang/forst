package nodeinterop

import (
	"fmt"
)

// BuildManifestV1 builds a forst-node-manifest-v1 from boundary root and per-module index data.
func BuildManifestV1(boundaryRoot string, indexes []ForstIndexV1) (ManifestV1, error) {
	ptrs := make([]*IndexV1, 0, len(indexes))
	for i := range indexes {
		idx := indexes[i]
		ptrs = append(ptrs, &idx)
	}
	m, err := ManifestFromIndexes(boundaryRoot, ptrs)
	if err != nil {
		return ManifestV1{}, err
	}
	if m == nil {
		return ManifestV1{}, fmt.Errorf("manifest: nil result")
	}
	return *m, nil
}
