package maps

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"

	"cs2-demo-service/pkg/navigation"
)

// LoadNavigationCandidate reads only the NAV; it never loads/creates geometry.
// Format support is not evidence that this local asset was used by a demo.
func LoadNavigationCandidate(path, reference, expectedSHA string) (*navigation.Graph, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 16 || binary.LittleEndian.Uint32(raw[:4]) != 0xfeedface || binary.LittleEndian.Uint32(raw[4:8]) != 35 || binary.LittleEndian.Uint32(raw[8:12]) != 1 {
		return nil, fmt.Errorf("support navigation requires the audited Source 2 v35 subversion 1 format")
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(raw))
	if checksum != expectedSHA {
		return nil, fmt.Errorf("support NAV asset hash mismatch")
	}
	mesh, err := LoadNavMesh(path)
	if err != nil {
		return nil, err
	}
	after, err := os.ReadFile(path)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(after)) != checksum {
		return nil, fmt.Errorf("support NAV changed during load")
	}
	m := &MapManager{currentNav: mesh, inputs: map[string]InputProvenance{"nav_mesh": {RelativePath: reference, SHA256: checksum, Used: true}}}
	return m.NavigationGraphSnapshot()
}
