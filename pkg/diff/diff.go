package diff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ctm/cluster-time-machine/pkg/store"
	"gopkg.in/yaml.v3"
)

// JSONToCleanYAML converts a JSON string to formatted YAML
func JSONToCleanYAML(jsonStr string) (string, error) {
	if strings.TrimSpace(jsonStr) == "" {
		return "", nil
	}
	var obj any
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ComputeDiff compares old and new JSON snapshots and produces a structured RevisionDiff
func ComputeDiff(oldRev, newRev *store.ResourceRevision) (*store.RevisionDiff, error) {
	diff := &store.RevisionDiff{
		OldRevision: oldRev,
		NewRevision: newRev,
	}

	var oldMap, newMap map[string]any

	if oldRev != nil && oldRev.RawJSON != "" {
		_ = json.Unmarshal([]byte(oldRev.RawJSON), &oldMap)
		oldYAML, err := JSONToCleanYAML(oldRev.RawJSON)
		if err == nil {
			diff.OldYAML = oldYAML
		}
	}

	if newRev != nil && newRev.RawJSON != "" {
		_ = json.Unmarshal([]byte(newRev.RawJSON), &newMap)
		newYAML, err := JSONToCleanYAML(newRev.RawJSON)
		if err == nil {
			diff.NewYAML = newYAML
		}
	}

	diff.FieldChanges = CompareMaps("", oldMap, newMap)
	diff.UnifiedDiff = generateUnifiedDiff(diff.OldYAML, diff.NewYAML)

	return diff, nil
}

// CompareMaps recursively traverses two maps and lists field differences
func CompareMaps(prefix string, oldMap, newMap map[string]any) []store.FieldDifference {
	var diffs []store.FieldDifference

	allKeys := make(map[string]bool)
	for k := range oldMap {
		allKeys[k] = true
	}
	for k := range newMap {
		allKeys[k] = true
	}

	sortedKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, k := range sortedKeys {
		if k == "managedFields" {
			continue
		}

		currentPath := k
		if prefix != "" {
			currentPath = prefix + "." + k
		}

		oldVal, oldExists := oldMap[k]
		newVal, newExists := newMap[k]

		if !oldExists && newExists {
			diffs = append(diffs, store.FieldDifference{
				Path:     currentPath,
				OldValue: nil,
				NewValue: newVal,
				Type:     "added",
			})
			continue
		}

		if oldExists && !newExists {
			diffs = append(diffs, store.FieldDifference{
				Path:     currentPath,
				OldValue: oldVal,
				NewValue: nil,
				Type:     "removed",
			})
			continue
		}

		oldSubMap, oldIsMap := oldVal.(map[string]any)
		newSubMap, newIsMap := newVal.(map[string]any)
		if oldIsMap && newIsMap {
			diffs = append(diffs, CompareMaps(currentPath, oldSubMap, newSubMap)...)
			continue
		}

		oldSlice, oldIsSlice := oldVal.([]any)
		newSlice, newIsSlice := newVal.([]any)
		if oldIsSlice && newIsSlice {
			diffs = append(diffs, compareSlices(currentPath, oldSlice, newSlice)...)
			continue
		}

		if !reflect.DeepEqual(oldVal, newVal) {
			diffs = append(diffs, store.FieldDifference{
				Path:     currentPath,
				OldValue: oldVal,
				NewValue: newVal,
				Type:     "modified",
			})
		}
	}

	return diffs
}

func compareSlices(path string, oldSlice, newSlice []any) []store.FieldDifference {
	var diffs []store.FieldDifference

	maxLen := len(oldSlice)
	if len(newSlice) > maxLen {
		maxLen = len(newSlice)
	}

	for i := 0; i < maxLen; i++ {
		idxPath := fmt.Sprintf("%s[%d]", path, i)
		if i >= len(oldSlice) {
			diffs = append(diffs, store.FieldDifference{
				Path:     idxPath,
				OldValue: nil,
				NewValue: newSlice[i],
				Type:     "added",
			})
		} else if i >= len(newSlice) {
			diffs = append(diffs, store.FieldDifference{
				Path:     idxPath,
				OldValue: oldSlice[i],
				NewValue: nil,
				Type:     "removed",
			})
		} else {
			oldItem := oldSlice[i]
			newItem := newSlice[i]

			oldItemMap, oldIsMap := oldItem.(map[string]any)
			newItemMap, newIsMap := newItem.(map[string]any)
			if oldIsMap && newIsMap {
				diffs = append(diffs, CompareMaps(idxPath, oldItemMap, newItemMap)...)
			} else if !reflect.DeepEqual(oldItem, newItem) {
				diffs = append(diffs, store.FieldDifference{
					Path:     idxPath,
					OldValue: oldItem,
					NewValue: newItem,
					Type:     "modified",
				})
			}
		}
	}

	return diffs
}

func generateUnifiedDiff(oldText, newText string) string {
	oldLines := strings.Split(strings.ReplaceAll(oldText, "\r\n", "\n"), "\n")
	newLines := strings.Split(strings.ReplaceAll(newText, "\r\n", "\n"), "\n")

	var sb strings.Builder
	sb.WriteString("--- previous\n+++ current\n")

	i, j := 0, 0
	for i < len(oldLines) && j < len(newLines) {
		if oldLines[i] == newLines[j] {
			sb.WriteString("  " + oldLines[i] + "\n")
			i++
			j++
		} else {
			sb.WriteString("- " + oldLines[i] + "\n")
			sb.WriteString("+ " + newLines[j] + "\n")
			i++
			j++
		}
	}

	for ; i < len(oldLines); i++ {
		if strings.TrimSpace(oldLines[i]) != "" {
			sb.WriteString("- " + oldLines[i] + "\n")
		}
	}

	for ; j < len(newLines); j++ {
		if strings.TrimSpace(newLines[j]) != "" {
			sb.WriteString("+ " + newLines[j] + "\n")
		}
	}

	return sb.String()
}

func SummarizeChanges(fieldDiffs []store.FieldDifference) string {
	if len(fieldDiffs) == 0 {
		return "Metadata/routine status sync"
	}

	var highlights []string
	for _, fd := range fieldDiffs {
		if strings.Contains(fd.Path, "resources.limits") || strings.Contains(fd.Path, "resources.requests") {
			highlights = append(highlights, fmt.Sprintf("%s: %v → %v", fd.Path, fd.OldValue, fd.NewValue))
		} else if strings.Contains(fd.Path, "image") {
			highlights = append(highlights, fmt.Sprintf("Image changed: %v → %v", fd.OldValue, fd.NewValue))
		} else if strings.Contains(fd.Path, "replicas") {
			highlights = append(highlights, fmt.Sprintf("Replicas changed: %v → %v", fd.OldValue, fd.NewValue))
		} else if strings.Contains(fd.Path, "env") {
			highlights = append(highlights, fmt.Sprintf("Env modified: %s", fd.Path))
		} else if strings.Contains(fd.Path, "nodeSelector") || strings.Contains(fd.Path, "tolerations") {
			highlights = append(highlights, fmt.Sprintf("Scheduling policy updated: %s", fd.Path))
		}
	}

	if len(highlights) > 0 {
		return strings.Join(highlights, "; ")
	}

	if len(fieldDiffs) == 1 {
		fd := fieldDiffs[0]
		return fmt.Sprintf("%s %s (%v → %v)", fd.Type, fd.Path, fd.OldValue, fd.NewValue)
	}

	return fmt.Sprintf("%d field(s) modified: %s", len(fieldDiffs), fieldDiffs[0].Path)
}
