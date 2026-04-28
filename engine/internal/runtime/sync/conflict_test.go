package sync

import (
	"testing"
)

func TestResolveFieldConflicts_NoConflict(t *testing.T) {
	base := map[string]interface{}{"name": "Alice", "email": "a@b.com", "phone": "123"}
	local := map[string]interface{}{"name": "Alice", "email": "a@b.com", "phone": "123"}
	remote := map[string]interface{}{"name": "Alice", "email": "a@b.com", "phone": "123"}

	result := ResolveFieldConflicts(base, local, remote, "", "")

	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(result.Conflicts))
	}
	if result.Resolution != ResolutionAutoMerge {
		t.Errorf("expected AUTO_MERGE, got %s", result.Resolution)
	}
}

func TestResolveFieldConflicts_DifferentFields(t *testing.T) {
	base := map[string]interface{}{"name": "Alice", "email": "a@b.com", "phone": "123"}
	local := map[string]interface{}{"name": "Alice Updated", "email": "a@b.com", "phone": "123"}
	remote := map[string]interface{}{"name": "Alice", "email": "new@b.com", "phone": "123"}

	result := ResolveFieldConflicts(base, local, remote, "", "")

	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts (auto-merge), got %d", len(result.Conflicts))
	}
	if result.MergedData["name"] != "Alice Updated" {
		t.Errorf("expected local name, got %v", result.MergedData["name"])
	}
	if result.MergedData["email"] != "new@b.com" {
		t.Errorf("expected remote email, got %v", result.MergedData["email"])
	}
}

func TestResolveFieldConflicts_SameFieldHLCWins(t *testing.T) {
	base := map[string]interface{}{"price": float64(100)}
	local := map[string]interface{}{"price": float64(150)}
	remote := map[string]interface{}{"price": float64(120)}

	localHLC := "zzzzzz:0001:DEV-A"
	remoteHLC := "aaaaaa:0001:DEV-B"

	result := ResolveFieldConflicts(base, local, remote, localHLC, remoteHLC)

	if len(result.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(result.Conflicts))
	}

	if result.MergedData["price"] != float64(150) {
		t.Errorf("expected local price (150) to win via HLC, got %v", result.MergedData["price"])
	}

	if result.Conflicts[0].Resolution != string(ResolutionLocalWins) {
		t.Errorf("expected LOCAL_WINS, got %s", result.Conflicts[0].Resolution)
	}
}

func TestResolveFieldConflicts_RemoteHLCWins(t *testing.T) {
	base := map[string]interface{}{"price": float64(100)}
	local := map[string]interface{}{"price": float64(150)}
	remote := map[string]interface{}{"price": float64(120)}

	localHLC := "aaaaaa:0001:DEV-A"
	remoteHLC := "zzzzzz:0001:DEV-B"

	result := ResolveFieldConflicts(base, local, remote, localHLC, remoteHLC)

	if result.MergedData["price"] != float64(120) {
		t.Errorf("expected remote price (120) to win via HLC, got %v", result.MergedData["price"])
	}

	if result.Conflicts[0].Resolution != string(ResolutionRemoteWins) {
		t.Errorf("expected REMOTE_WINS, got %s", result.Conflicts[0].Resolution)
	}
}

func TestResolveFieldConflicts_SameValueNoConflict(t *testing.T) {
	base := map[string]interface{}{"name": "Alice"}
	local := map[string]interface{}{"name": "Bob"}
	remote := map[string]interface{}{"name": "Bob"}

	result := ResolveFieldConflicts(base, local, remote, "", "")

	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts when both changed to same value, got %d", len(result.Conflicts))
	}
	if result.MergedData["name"] != "Bob" {
		t.Errorf("expected 'Bob', got %v", result.MergedData["name"])
	}
}

func TestResolveFieldConflicts_SkipsSystemFields(t *testing.T) {
	base := map[string]interface{}{"id": "abc", "_off_version": 1, "_off_status": "SYNCED", "name": "Alice"}
	local := map[string]interface{}{"id": "abc", "_off_version": 3, "_off_status": "PENDING", "name": "Alice"}
	remote := map[string]interface{}{"id": "abc", "_off_version": 2, "_off_status": "SYNCED", "name": "Alice"}

	result := ResolveFieldConflicts(base, local, remote, "", "")

	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts (system fields skipped), got %d", len(result.Conflicts))
	}
	if _, ok := result.MergedData["id"]; ok {
		t.Error("system field 'id' should not be in merged data")
	}
}

func TestResolveFieldConflicts_HLCTieBreakByDeviceID(t *testing.T) {
	base := map[string]interface{}{"qty": float64(10)}
	local := map[string]interface{}{"qty": float64(5)}
	remote := map[string]interface{}{"qty": float64(8)}

	hlcA := "abc:0001:DEV-A"
	hlcB := "abc:0001:DEV-B"

	result := ResolveFieldConflicts(base, local, remote, hlcA, hlcB)

	if len(result.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(result.Conflicts))
	}

	// DEV-A < DEV-B lexicographically, so DEV-B (remote) wins
	if result.MergedData["qty"] != float64(8) {
		t.Errorf("expected remote qty (8) to win via device_id tie-break, got %v", result.MergedData["qty"])
	}
}

func TestResolveEditVsDelete_EditWins(t *testing.T) {
	editData := map[string]interface{}{"name": "Updated", "price": float64(200)}

	result := ResolveEditVsDelete(editData, true)

	if result.Resolution != ResolutionEditWins {
		t.Errorf("expected EDIT_WINS, got %s", result.Resolution)
	}

	if result.MergedData["_off_deleted"] != 0 {
		t.Error("expected _off_deleted = 0 (resurrected)")
	}

	if result.MergedData["name"] != "Updated" {
		t.Errorf("expected edit data preserved, got %v", result.MergedData["name"])
	}

	if len(result.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict entry, got %d", len(result.Conflicts))
	}
}

func TestResolveByHLC_EmptyStrings(t *testing.T) {
	if resolveByHLC("", "") != 0 {
		t.Error("expected 0 for both empty")
	}
	if resolveByHLC("abc:0001:DEV-A", "") != 1 {
		t.Error("expected 1 when remote is empty")
	}
	if resolveByHLC("", "abc:0001:DEV-B") != -1 {
		t.Error("expected -1 when local is empty")
	}
}

func TestParseBase36(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0", 0},
		{"1", 1},
		{"a", 10},
		{"z", 35},
		{"10", 36},
		{"zz", 1295},
	}

	for _, tt := range tests {
		got := parseBase36(tt.input)
		if got != tt.expected {
			t.Errorf("parseBase36(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestValuesEqual(t *testing.T) {
	if !valuesEqual(nil, nil) {
		t.Error("nil == nil should be true")
	}
	if valuesEqual(nil, "x") {
		t.Error("nil != 'x'")
	}
	if !valuesEqual("hello", "hello") {
		t.Error("same strings should be equal")
	}
	if !valuesEqual(float64(42), float64(42)) {
		t.Error("same numbers should be equal")
	}
	if valuesEqual(float64(42), float64(43)) {
		t.Error("different numbers should not be equal")
	}
}

func TestIsSystemField(t *testing.T) {
	if !isSystemField("id") {
		t.Error("'id' should be system field")
	}
	if !isSystemField("_off_version") {
		t.Error("'_off_version' should be system field")
	}
	if !isSystemField("_sync_log") {
		t.Error("'_sync_log' should be system field")
	}
	if isSystemField("name") {
		t.Error("'name' should not be system field")
	}
	if isSystemField("price") {
		t.Error("'price' should not be system field")
	}
}
