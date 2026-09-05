package stage2enricher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestActorStore_ReadWriteAndMerge(t *testing.T) {
	tmpDir := t.TempDir()
	actorFile := filepath.Join(tmpDir, "actor.json")

	store := NewActorStore(actorFile, "", nil)

	// 1. Initially empty
	_, found, err := store.FindDir([]string{"Yui Hatano"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatalf("expected not found initially")
	}

	// 2. Add actor
	chosen, err := store.AddActorAlias("波多野结衣", []string{"Yui Hatano", "波多野结衣", "波多野結衣"})
	if err != nil {
		t.Fatalf("AddActorAlias failed: %v", err)
	}
	if chosen != "波多野结衣" {
		t.Fatalf("expected chosen 波多野结衣, got %s", chosen)
	}

	// 3. FindDir by alias
	dir, found, err := store.FindDir([]string{"Yui Hatano"})
	if err != nil || !found || dir != "波多野结衣" {
		t.Fatalf("expected found 波多野结衣, got %s (%t)", dir, found)
	}

	// 4. Merge new alias into existing directory
	chosen2, err := store.AddActorAlias("波多野_other", []string{"波多野結衣", "Hatano Yui New"})
	if err != nil {
		t.Fatalf("AddActorAlias merge failed: %v", err)
	}
	if chosen2 != "波多野结衣" {
		t.Fatalf("expected merged into 波多野结衣, got %s", chosen2)
	}

	// Verify disk format
	data, err := os.ReadFile(actorFile)
	if err != nil {
		t.Fatalf("failed to read actor.json: %v", err)
	}
	var diskMap map[string][]string
	if err := json.Unmarshal(data, &diskMap); err != nil {
		t.Fatalf("failed to parse disk json: %v", err)
	}
	if len(diskMap["波多野结衣"]) < 4 {
		t.Fatalf("expected merged aliases, got: %v", diskMap["波多野结衣"])
	}
}

func TestActorStore_FlockConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	actorFile := filepath.Join(tmpDir, "actor.json")

	store := NewActorStore(actorFile, "", nil)

	var wg sync.WaitGroup
	workers := 10
	iterations := 5

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				name := "ActorA"
				if workerID%2 == 1 {
					name = "ActorB"
				}
				alias := name + "_alias"
				_, err := store.AddActorAlias(name, []string{alias})
				if err != nil {
					t.Errorf("concurrent AddActorAlias failed: %v", err)
				}
			}
		}(w)
	}

	wg.Wait()

	// Verify both exist and file is valid JSON
	data, err := os.ReadFile(actorFile)
	if err != nil {
		t.Fatalf("failed to read actor file: %v", err)
	}
	var res map[string][]string
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("corrupted JSON from concurrent writes: %v", err)
	}
	if _, ok := res["ActorA"]; !ok {
		t.Fatalf("missing ActorA")
	}
	if _, ok := res["ActorB"]; !ok {
		t.Fatalf("missing ActorB")
	}
}

func TestActorStore_ParseArchived1Json(t *testing.T) {
	// Test extracting aliases from archived/1.json (real FlareSolverr HTML)
	paths := []string{
		"../../../archived/1.json",
		"../../archived/1.json",
		"archived/1.json",
		"/home/user/src/autoget-project/organizer/archived/1.json",
	}
	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Skipf("archived/1.json not found: %v", err)
	}

	aliases := ParseJavDBResponse(data, "三上悠亚")
	if len(aliases) == 0 {
		t.Fatalf("expected aliases extracted from archived/1.json")
	}

	// Verify 三上悠亜, 鬼头桃菜 are extracted from title="三上悠亞, 鬼头桃菜"
	foundMikami := false
	foundKito := false
	for _, a := range aliases {
		if a == "三上悠亞" || a == "三上悠亚" {
			foundMikami = true
		}
		if a == "鬼头桃菜" {
			foundKito = true
		}
	}

	if !foundMikami || !foundKito {
		t.Fatalf("expected 三上悠亞 and 鬼头桃菜 in aliases, got: %v", aliases)
	}
}
