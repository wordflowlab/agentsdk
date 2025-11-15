package memory

import (
	"context"
	"testing"
	"time"
)

func TestLineageGraph_TrackMemory(t *testing.T) {
	lg := NewLineageGraph()

	metadata := &LineageMetadata{
		ID:             "mem-1",
		SourceIDs:      []string{"session-1"},
		DerivedFromIDs: []string{},
		CreatedAt:      time.Now().Unix(),
	}

	lg.TrackMemory("mem-1", metadata)

	if lg.memoryMetadata["mem-1"] == nil {
		t.Error("Memory metadata not tracked")
	}

	if lg.memoryMetadata["mem-1"].ID != "mem-1" {
		t.Errorf("Memory ID = %v, want mem-1", lg.memoryMetadata["mem-1"].ID)
	}
}

func TestLineageGraph_GetDerivedMemories(t *testing.T) {
	lg := NewLineageGraph()

	// Create a lineage tree:
	// mem-1
	//  ├─ mem-2
	//  └─ mem-3
	//      └─ mem-4

	lg.TrackMemory("mem-1", &LineageMetadata{
		ID:             "mem-1",
		SourceIDs:      []string{"session-1"},
		DerivedFromIDs: []string{},
		CreatedAt:      time.Now().Unix(),
	})

	lg.TrackMemory("mem-2", &LineageMetadata{
		ID:             "mem-2",
		SourceIDs:      []string{"session-1"},
		DerivedFromIDs: []string{"mem-1"},
		CreatedAt:      time.Now().Unix(),
	})

	lg.TrackMemory("mem-3", &LineageMetadata{
		ID:             "mem-3",
		SourceIDs:      []string{"session-1"},
		DerivedFromIDs: []string{"mem-1"},
		CreatedAt:      time.Now().Unix(),
	})

	lg.TrackMemory("mem-4", &LineageMetadata{
		ID:             "mem-4",
		SourceIDs:      []string{"session-1"},
		DerivedFromIDs: []string{"mem-3"},
		CreatedAt:      time.Now().Unix(),
	})

	// Get all derived from mem-1
	derived := lg.GetDerivedMemories("mem-1")

	// Should include mem-2, mem-3, mem-4
	if len(derived) != 3 {
		t.Fatalf("GetDerivedMemories() returned %d memories, want 3", len(derived))
	}

	// Check all expected IDs are present
	expectedIDs := map[string]bool{
		"mem-2": false,
		"mem-3": false,
		"mem-4": false,
	}

	for _, id := range derived {
		if _, exists := expectedIDs[id]; exists {
			expectedIDs[id] = true
		} else {
			t.Errorf("Unexpected derived memory: %v", id)
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Expected derived memory %v not found", id)
		}
	}

	// Get derived from mem-3 (should only be mem-4)
	derived = lg.GetDerivedMemories("mem-3")
	if len(derived) != 1 || derived[0] != "mem-4" {
		t.Errorf("GetDerivedMemories('mem-3') = %v, want [mem-4]", derived)
	}

	// Get derived from mem-2 (should be empty)
	derived = lg.GetDerivedMemories("mem-2")
	if len(derived) != 0 {
		t.Errorf("GetDerivedMemories('mem-2') should be empty, got %v", derived)
	}
}

func TestLineageGraph_GetParentMemories(t *testing.T) {
	lg := NewLineageGraph()

	// Create a lineage tree:
	// mem-1
	//  └─ mem-2
	//      └─ mem-3 (also derives from mem-4)
	// mem-4

	lg.TrackMemory("mem-1", &LineageMetadata{
		ID:             "mem-1",
		DerivedFromIDs: []string{},
		CreatedAt:      time.Now().Unix(),
	})

	lg.TrackMemory("mem-2", &LineageMetadata{
		ID:             "mem-2",
		DerivedFromIDs: []string{"mem-1"},
		CreatedAt:      time.Now().Unix(),
	})

	lg.TrackMemory("mem-4", &LineageMetadata{
		ID:             "mem-4",
		DerivedFromIDs: []string{},
		CreatedAt:      time.Now().Unix(),
	})

	lg.TrackMemory("mem-3", &LineageMetadata{
		ID:             "mem-3",
		DerivedFromIDs: []string{"mem-2", "mem-4"},
		CreatedAt:      time.Now().Unix(),
	})

	// Get all parents of mem-3
	parents := lg.GetParentMemories("mem-3")

	// Should include mem-2, mem-1, mem-4
	if len(parents) != 3 {
		t.Fatalf("GetParentMemories() returned %d memories, want 3: %v", len(parents), parents)
	}

	expectedIDs := map[string]bool{
		"mem-1": false,
		"mem-2": false,
		"mem-4": false,
	}

	for _, id := range parents {
		if _, exists := expectedIDs[id]; exists {
			expectedIDs[id] = true
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Expected parent memory %v not found", id)
		}
	}
}

func TestLineageGraph_GetMemoriesBySource(t *testing.T) {
	lg := NewLineageGraph()

	lg.TrackMemory("mem-1", &LineageMetadata{
		ID:        "mem-1",
		SourceIDs: []string{"session-1"},
		CreatedAt: time.Now().Unix(),
	})

	lg.TrackMemory("mem-2", &LineageMetadata{
		ID:        "mem-2",
		SourceIDs: []string{"session-1", "session-2"},
		CreatedAt: time.Now().Unix(),
	})

	lg.TrackMemory("mem-3", &LineageMetadata{
		ID:        "mem-3",
		SourceIDs: []string{"session-2"},
		CreatedAt: time.Now().Unix(),
	})

	// Get memories from session-1
	memories := lg.GetMemoriesBySource("session-1")

	if len(memories) != 2 {
		t.Fatalf("GetMemoriesBySource('session-1') returned %d memories, want 2", len(memories))
	}

	// Get memories from session-2
	memories = lg.GetMemoriesBySource("session-2")

	if len(memories) != 2 {
		t.Fatalf("GetMemoriesBySource('session-2') returned %d memories, want 2", len(memories))
	}

	// Get memories from non-existent source
	memories = lg.GetMemoriesBySource("session-999")

	if len(memories) != 0 {
		t.Errorf("GetMemoriesBySource('session-999') should be empty, got %v", memories)
	}
}

func TestLineageGraph_RemoveMemory(t *testing.T) {
	lg := NewLineageGraph()

	// Create parent-child relationship
	lg.TrackMemory("parent", &LineageMetadata{
		ID:        "parent",
		CreatedAt: time.Now().Unix(),
	})

	lg.TrackMemory("child", &LineageMetadata{
		ID:             "child",
		DerivedFromIDs: []string{"parent"},
		CreatedAt:      time.Now().Unix(),
	})

	// Remove child
	lg.RemoveMemory("child")

	// Child should be gone
	if lg.memoryMetadata["child"] != nil {
		t.Error("Child memory should be removed from metadata")
	}

	// Parent should no longer have child in its children list
	if len(lg.parentToChildren["parent"]) != 0 {
		t.Error("Parent should have no children after removal")
	}
}

func TestLineageManager_TrackMemoryCreation(t *testing.T) {
	lm := NewLineageManager()

	provenance := NewProvenance(SourceUserInput, "session-1")

	err := lm.TrackMemoryCreation("mem-1", provenance, nil)
	if err != nil {
		t.Fatalf("TrackMemoryCreation() error = %v", err)
	}

	// Verify memory was tracked
	metadata := lm.graph.memoryMetadata["mem-1"]
	if metadata == nil {
		t.Fatal("Memory not tracked in graph")
	}

	if metadata.ID != "mem-1" {
		t.Errorf("Memory ID = %v, want mem-1", metadata.ID)
	}

	if len(metadata.SourceIDs) != 1 || metadata.SourceIDs[0] != "session-1" {
		t.Errorf("SourceIDs = %v, want [session-1]", metadata.SourceIDs)
	}
}

func TestLineageManager_DeleteMemoryWithLineage(t *testing.T) {
	lm := NewLineageManager()

	// Create lineage
	p1 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-1", p1, nil)

	p2 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-2", p2, []string{"mem-1"})

	p3 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-3", p3, []string{"mem-2"})

	ctx := context.Background()

	// Delete with cascade
	deleted, err := lm.DeleteMemoryWithLineage(ctx, "mem-1", true)
	if err != nil {
		t.Fatalf("DeleteMemoryWithLineage() error = %v", err)
	}

	// Should delete mem-1, mem-2, mem-3
	if len(deleted) != 3 {
		t.Fatalf("Deleted %d memories, want 3", len(deleted))
	}

	expectedDeleted := map[string]bool{
		"mem-1": false,
		"mem-2": false,
		"mem-3": false,
	}

	for _, id := range deleted {
		if _, exists := expectedDeleted[id]; exists {
			expectedDeleted[id] = true
		}
	}

	for id, found := range expectedDeleted {
		if !found {
			t.Errorf("Expected memory %v to be deleted", id)
		}
	}

	// Verify memories are removed from graph
	if lm.graph.memoryMetadata["mem-1"] != nil {
		t.Error("mem-1 should be removed from graph")
	}
}

func TestLineageManager_DeleteMemoryWithoutCascade(t *testing.T) {
	lm := NewLineageManager()

	// Create lineage
	p1 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-1", p1, nil)

	p2 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-2", p2, []string{"mem-1"})

	ctx := context.Background()

	// Delete without cascade
	deleted, err := lm.DeleteMemoryWithLineage(ctx, "mem-1", false)
	if err != nil {
		t.Fatalf("DeleteMemoryWithLineage() error = %v", err)
	}

	// Should only delete mem-1
	if len(deleted) != 1 || deleted[0] != "mem-1" {
		t.Errorf("Deleted = %v, want [mem-1]", deleted)
	}

	// mem-2 should still exist
	if lm.graph.memoryMetadata["mem-2"] == nil {
		t.Error("mem-2 should still exist in graph")
	}
}

func TestLineageManager_RevokeDataSource(t *testing.T) {
	lm := NewLineageManager()

	// Create memories from different sources
	p1 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-1", p1, nil)

	p2 := NewProvenance(SourceUserInput, "session-2")
	lm.TrackMemoryCreation("mem-2", p2, nil)

	p3 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-3", p3, []string{"mem-1"})

	ctx := context.Background()

	// Revoke session-1
	deleted, err := lm.RevokeDataSource(ctx, "session-1")
	if err != nil {
		t.Fatalf("RevokeDataSource() error = %v", err)
	}

	// Should delete mem-1 and mem-3 (derived from mem-1)
	if len(deleted) < 2 {
		t.Errorf("Deleted %d memories, want at least 2", len(deleted))
	}

	// mem-2 should still exist
	if lm.graph.memoryMetadata["mem-2"] == nil {
		t.Error("mem-2 should still exist (different source)")
	}
}

func TestLineageManager_GetLineageDepth(t *testing.T) {
	lm := NewLineageManager()

	// Create a chain: mem-1 -> mem-2 -> mem-3 -> mem-4
	p1 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-1", p1, nil)

	p2 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-2", p2, []string{"mem-1"})

	p3 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-3", p3, []string{"mem-2"})

	p4 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-4", p4, []string{"mem-3"})

	tests := []struct {
		memoryID  string
		wantDepth int
	}{
		{"mem-1", 0},
		{"mem-2", 1},
		{"mem-3", 2},
		{"mem-4", 3},
	}

	for _, tt := range tests {
		t.Run(tt.memoryID, func(t *testing.T) {
			depth := lm.GetLineageDepth(tt.memoryID)
			if depth != tt.wantDepth {
				t.Errorf("GetLineageDepth(%v) = %v, want %v", tt.memoryID, depth, tt.wantDepth)
			}
		})
	}
}

func TestLineageManager_GetLineageStats(t *testing.T) {
	lm := NewLineageManager()

	// Create some memories
	p1 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-1", p1, nil) // Root

	p2 := NewProvenance(SourceUserInput, "session-1")
	lm.TrackMemoryCreation("mem-2", p2, []string{"mem-1"}) // Derived

	p3 := NewProvenance(SourceUserInput, "session-2")
	lm.TrackMemoryCreation("mem-3", p3, nil) // Root

	stats := lm.GetLineageStats()

	if stats.TotalMemories != 3 {
		t.Errorf("TotalMemories = %v, want 3", stats.TotalMemories)
	}

	if stats.RootMemories != 2 {
		t.Errorf("RootMemories = %v, want 2", stats.RootMemories)
	}

	if stats.DerivedMemories != 1 {
		t.Errorf("DerivedMemories = %v, want 1", stats.DerivedMemories)
	}

	if stats.MaxDepth != 1 {
		t.Errorf("MaxDepth = %v, want 1", stats.MaxDepth)
	}

	if stats.MemoriesBySource["session-1"] != 2 {
		t.Errorf("MemoriesBySource['session-1'] = %v, want 2", stats.MemoriesBySource["session-1"])
	}

	if stats.MemoriesBySource["session-2"] != 1 {
		t.Errorf("MemoriesBySource['session-2'] = %v, want 1", stats.MemoriesBySource["session-2"])
	}
}
