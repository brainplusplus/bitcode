package sync

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ConflictResolution string

const (
	ResolutionLocalWins  ConflictResolution = "LOCAL_WINS"
	ResolutionRemoteWins ConflictResolution = "REMOTE_WINS"
	ResolutionAutoMerge  ConflictResolution = "AUTO_MERGE"
	ResolutionEditWins   ConflictResolution = "EDIT_WINS"
)

type FieldConflict struct {
	FieldName     string `json:"field_name"`
	LocalValue    string `json:"local_value"`
	RemoteValue   string `json:"remote_value"`
	ResolvedValue string `json:"resolved_value"`
	Resolution    string `json:"resolution"`
	LocalHLC      string `json:"local_hlc,omitempty"`
	RemoteHLC     string `json:"remote_hlc,omitempty"`
}

type MergeResult struct {
	MergedData map[string]interface{}
	Conflicts  []FieldConflict
	Resolution ConflictResolution
}

// ResolveFieldConflicts performs field-level merge between local and remote records.
// baseData is the last-synced snapshot (common ancestor), localData is the current local state,
// remoteData is the incoming server state. localHLC/remoteHLC are used for same-field tie-breaking.
func ResolveFieldConflicts(
	baseData map[string]interface{},
	localData map[string]interface{},
	remoteData map[string]interface{},
	localHLC string,
	remoteHLC string,
) MergeResult {
	merged := make(map[string]interface{})
	var conflicts []FieldConflict

	allFields := collectFieldNames(baseData, localData, remoteData)

	for _, field := range allFields {
		if isSystemField(field) {
			continue
		}

		baseVal := baseData[field]
		localVal := localData[field]
		remoteVal := remoteData[field]

		localChanged := !valuesEqual(baseVal, localVal)
		remoteChanged := !valuesEqual(baseVal, remoteVal)

		switch {
		case !localChanged && !remoteChanged:
			merged[field] = localVal

		case !localChanged && remoteChanged:
			merged[field] = remoteVal

		case localChanged && !remoteChanged:
			merged[field] = localVal

		case localChanged && remoteChanged:
			if valuesEqual(localVal, remoteVal) {
				merged[field] = localVal
				continue
			}

			winner := resolveByHLC(localHLC, remoteHLC)
			var resolvedVal interface{}
			var resolution string

			if winner >= 0 {
				resolvedVal = localVal
				resolution = string(ResolutionLocalWins)
			} else {
				resolvedVal = remoteVal
				resolution = string(ResolutionRemoteWins)
			}

			merged[field] = resolvedVal
			conflicts = append(conflicts, FieldConflict{
				FieldName:     field,
				LocalValue:    fmt.Sprintf("%v", localVal),
				RemoteValue:   fmt.Sprintf("%v", remoteVal),
				ResolvedValue: fmt.Sprintf("%v", resolvedVal),
				Resolution:    resolution,
				LocalHLC:      localHLC,
				RemoteHLC:     remoteHLC,
			})
		}
	}

	overallResolution := ResolutionAutoMerge
	if len(conflicts) > 0 {
		hasLocal := false
		hasRemote := false
		for _, c := range conflicts {
			if c.Resolution == string(ResolutionLocalWins) {
				hasLocal = true
			}
			if c.Resolution == string(ResolutionRemoteWins) {
				hasRemote = true
			}
		}
		if hasLocal && !hasRemote {
			overallResolution = ResolutionLocalWins
		} else if hasRemote && !hasLocal {
			overallResolution = ResolutionRemoteWins
		}
	}

	return MergeResult{
		MergedData: merged,
		Conflicts:  conflicts,
		Resolution: overallResolution,
	}
}

// ResolveEditVsDelete handles the case where one device edits and another deletes.
// Per design doc: edit wins — deleted record is resurrected with the edit applied.
func ResolveEditVsDelete(
	editData map[string]interface{},
	isLocalEdit bool,
) MergeResult {
	merged := make(map[string]interface{})
	for k, v := range editData {
		merged[k] = v
	}
	merged["_off_deleted"] = 0

	resolution := ResolutionEditWins
	var conflicts []FieldConflict

	source := "REMOTE"
	if isLocalEdit {
		source = "LOCAL"
	}

	conflicts = append(conflicts, FieldConflict{
		FieldName:     "_off_deleted",
		LocalValue:    boolStr(isLocalEdit, "0", "1"),
		RemoteValue:   boolStr(isLocalEdit, "1", "0"),
		ResolvedValue: "0",
		Resolution:    fmt.Sprintf("EDIT_WINS_%s", source),
	})

	return MergeResult{
		MergedData: merged,
		Conflicts:  conflicts,
		Resolution: resolution,
	}
}

// RecordConflictsToServer inserts conflict entries into _sync_conflicts for admin review.
func RecordConflictsToServer(
	db *gorm.DB,
	envelopeID string,
	deviceID string,
	otherDeviceID string,
	tableName string,
	recordID string,
	conflicts []FieldConflict,
) error {
	if len(conflicts) == 0 {
		return nil
	}

	now := time.Now().UTC()

	for _, c := range conflicts {
		err := db.Exec(
			`INSERT INTO _sync_conflicts (envelope_id, device_id, other_device_id, table_name, record_id, field_name, device_value, server_value, resolved_value, resolution, auto_resolved, created_at, device_hlc, server_hlc)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			envelopeID, deviceID, otherDeviceID, tableName, recordID,
			c.FieldName, c.LocalValue, c.RemoteValue, c.ResolvedValue,
			c.Resolution, true, now, c.LocalHLC, c.RemoteHLC,
		).Error
		if err != nil {
			return fmt.Errorf("failed to record conflict for field %s: %w", c.FieldName, err)
		}
	}

	return nil
}

func collectFieldNames(maps ...map[string]interface{}) []string {
	seen := make(map[string]bool)
	var result []string
	for _, m := range maps {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				result = append(result, k)
			}
		}
	}
	return result
}

var systemFieldPrefixes = []string{"_off_", "_sync_"}

func isSystemField(field string) bool {
	if field == "id" {
		return true
	}
	for _, prefix := range systemFieldPrefixes {
		if strings.HasPrefix(field, prefix) {
			return true
		}
	}
	return false
}

func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
	return string(aJSON) == string(bJSON)
}

// resolveByHLC compares two HLC strings.
// HLC format: "{wall_time_base36}:{logical_base36}:{device_id}"
// Returns: 1 if local wins, -1 if remote wins, 0 if tie.
func resolveByHLC(localHLC, remoteHLC string) int {
	if localHLC == "" && remoteHLC == "" {
		return 0
	}
	if localHLC == "" {
		return -1
	}
	if remoteHLC == "" {
		return 1
	}

	localParts := splitHLC(localHLC)
	remoteParts := splitHLC(remoteHLC)

	if localParts.wallTime > remoteParts.wallTime {
		return 1
	}
	if localParts.wallTime < remoteParts.wallTime {
		return -1
	}

	if localParts.logical > remoteParts.logical {
		return 1
	}
	if localParts.logical < remoteParts.logical {
		return -1
	}

	if localParts.deviceID > remoteParts.deviceID {
		return 1
	}
	if localParts.deviceID < remoteParts.deviceID {
		return -1
	}

	return 0
}

type hlcParts struct {
	wallTime int64
	logical  int64
	deviceID string
}

func splitHLC(hlc string) hlcParts {
	firstColon := strings.Index(hlc, ":")
	if firstColon == -1 {
		return hlcParts{}
	}
	secondColon := strings.Index(hlc[firstColon+1:], ":")
	if secondColon == -1 {
		return hlcParts{}
	}
	secondColon += firstColon + 1

	wallStr := hlc[:firstColon]
	logicalStr := hlc[firstColon+1 : secondColon]
	deviceID := hlc[secondColon+1:]

	wallTime := parseBase36(wallStr)
	logical := parseBase36(logicalStr)

	return hlcParts{wallTime: wallTime, logical: logical, deviceID: deviceID}
}

func parseBase36(s string) int64 {
	var result int64
	for _, c := range s {
		result *= 36
		switch {
		case c >= '0' && c <= '9':
			result += int64(c - '0')
		case c >= 'a' && c <= 'z':
			result += int64(c-'a') + 10
		case c >= 'A' && c <= 'Z':
			result += int64(c-'A') + 10
		}
	}
	return result
}

func boolStr(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}

var validIdentChars = func() [256]bool {
	var t [256]bool
	for c := 'a'; c <= 'z'; c++ {
		t[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		t[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		t[c] = true
	}
	t['_'] = true
	return t
}()

func isValidTableName(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !validIdentChars[name[i]] {
			return false
		}
	}
	return true
}
