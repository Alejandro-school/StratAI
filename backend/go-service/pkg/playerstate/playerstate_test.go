package playerstate

import "testing"

func TestEquipmentEntityIDIsStableAndNamespacesReusedEntities(t *testing.T) {
	first := EquipmentEntityID(42, 7)
	if first == 0 || first != EquipmentEntityID(42, 7) {
		t.Fatalf("stable entity identity = %d", first)
	}
	if first == EquipmentEntityID(42, 8) || first == EquipmentEntityID(43, 7) {
		t.Fatal("entity ID and serial number must both participate in identity")
	}
	if EquipmentEntityID(0, 7) != 0 || EquipmentEntityID(42, -1) != 0 {
		t.Fatal("unavailable native identity must remain explicit as zero")
	}
}
