package plumbing

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/hercules.v7/internal/core"
	"gopkg.in/src-d/hercules.v7/internal/test"
)

func fixtureRenameAnalysis() *RenameAnalysis {
	ra := RenameAnalysis{SimilarityThreshold: 80}
	ra.Initialize(test.Repository)
	return &ra
}

func TestRenameAnalysisMeta(t *testing.T) {
	ra := fixtureRenameAnalysis()
	assert.Equal(t, ra.Name(), "RenameAnalysis")
	assert.Equal(t, len(ra.Provides()), 1)
	assert.Equal(t, ra.Provides()[0], DependencyTreeChanges)
	assert.Equal(t, len(ra.Requires()), 2)
	assert.Equal(t, ra.Requires()[0], DependencyBlobCache)
	assert.Equal(t, ra.Requires()[1], DependencyTreeChanges)
	opts := ra.ListConfigurationOptions()
	assert.Len(t, opts, 1)
	assert.Equal(t, opts[0].Name, ConfigRenameAnalysisSimilarityThreshold)
	ra.SimilarityThreshold = 0
	facts := map[string]interface{}{}
	facts[ConfigRenameAnalysisSimilarityThreshold] = 70
	ra.Configure(facts)
	assert.Equal(t, ra.SimilarityThreshold, 70)
	delete(facts, ConfigRenameAnalysisSimilarityThreshold)
	ra.Configure(facts)
	assert.Equal(t, ra.SimilarityThreshold, 70)
}

func TestRenameAnalysisRegistration(t *testing.T) {
	summoned := core.Registry.Summon((&RenameAnalysis{}).Name())
	assert.Len(t, summoned, 1)
	assert.Equal(t, summoned[0].Name(), "RenameAnalysis")
	summoned = core.Registry.Summon((&RenameAnalysis{}).Provides()[0])
	assert.True(t, len(summoned) >= 1)
	matched := false
	for _, tp := range summoned {
		matched = matched || tp.Name() == "RenameAnalysis"
	}
	assert.True(t, matched)
}

func TestRenameAnalysisInitializeInvalidThreshold(t *testing.T) {
	ra := RenameAnalysis{SimilarityThreshold: -10}
	ra.Initialize(test.Repository)
	assert.Equal(t, ra.SimilarityThreshold, RenameAnalysisDefaultThreshold)
	ra = RenameAnalysis{SimilarityThreshold: 110}
	ra.Initialize(test.Repository)
	assert.Equal(t, ra.SimilarityThreshold, RenameAnalysisDefaultThreshold)
	ra = RenameAnalysis{SimilarityThreshold: 0}
	ra.Initialize(test.Repository)
	ra = RenameAnalysis{SimilarityThreshold: 100}
	ra.Initialize(test.Repository)
}

func TestRenameAnalysisConsume(t *testing.T) {
	ra := fixtureRenameAnalysis()
	changes := make(object.Changes, 3)
	// 2b1ed978194a94edeabbca6de7ff3b5771d4d665
	treeFrom, _ := test.Repository.TreeObject(plumbing.NewHash(
		"96c6ece9b2f3c7c51b83516400d278dea5605100"))
	treeTo, _ := test.Repository.TreeObject(plumbing.NewHash(
		"251f2094d7b523d5bcc60e663b6cf38151bf8844"))
	changes[0] = &object.Change{From: object.ChangeEntry{
		Name: "analyser.go",
		Tree: treeFrom,
		TreeEntry: object.TreeEntry{
			Name: "analyser.go",
			Mode: 0100644,
			Hash: plumbing.NewHash("baa64828831d174f40140e4b3cfa77d1e917a2c1"),
		},
	}, To: object.ChangeEntry{},
	}
	changes[1] = &object.Change{From: object.ChangeEntry{}, To: object.ChangeEntry{
		Name: "burndown.go",
		Tree: treeTo,
		TreeEntry: object.TreeEntry{
			Name: "burndown.go",
			Mode: 0100644,
			Hash: plumbing.NewHash("29c9fafd6a2fae8cd20298c3f60115bc31a4c0f2"),
		},
	},
	}
	changes[2] = &object.Change{From: object.ChangeEntry{
		Name: "cmd/hercules/main.go",
		Tree: treeFrom,
		TreeEntry: object.TreeEntry{
			Name: "cmd/hercules/main.go",
			Mode: 0100644,
			Hash: plumbing.NewHash("c29112dbd697ad9b401333b80c18a63951bc18d9"),
		},
	}, To: object.ChangeEntry{
		Name: "cmd/hercules/main.go",
		Tree: treeTo,
		TreeEntry: object.TreeEntry{
			Name: "cmd/hercules/main.go",
			Mode: 0100644,
			Hash: plumbing.NewHash("f7d918ec500e2f925ecde79b51cc007bac27de72"),
		},
	},
	}
	cache := map[plumbing.Hash]*CachedBlob{}
	AddHash(t, cache, "baa64828831d174f40140e4b3cfa77d1e917a2c1")
	AddHash(t, cache, "29c9fafd6a2fae8cd20298c3f60115bc31a4c0f2")
	AddHash(t, cache, "c29112dbd697ad9b401333b80c18a63951bc18d9")
	AddHash(t, cache, "f7d918ec500e2f925ecde79b51cc007bac27de72")
	deps := map[string]interface{}{}
	deps[DependencyBlobCache] = cache
	deps[DependencyTreeChanges] = changes
	ra.SimilarityThreshold = 37
	res, err := ra.Consume(deps)
	assert.Nil(t, err)
	renamed := res[DependencyTreeChanges].(object.Changes)
	assert.Equal(t, len(renamed), 2)
	ra.SimilarityThreshold = 39
	res, err = ra.Consume(deps)
	assert.Nil(t, err)
	renamed = res[DependencyTreeChanges].(object.Changes)
	assert.Equal(t, len(renamed), 3)
}

func TestSortableChanges(t *testing.T) {
	changes := sortableChanges{
		sortableChange{
			nil, plumbing.NewHash("0000000000000000000000000000000000000000"),
		}, sortableChange{
			nil, plumbing.NewHash("ffffffffffffffffffffffffffffffffffffffff"),
		},
	}
	assert.True(t, changes.Less(0, 1))
	assert.False(t, changes.Less(1, 0))
	assert.False(t, changes.Less(0, 0))
	changes.Swap(0, 1)
	assert.Equal(t, changes[0].hash.String(), "ffffffffffffffffffffffffffffffffffffffff")
	assert.Equal(t, changes[1].hash.String(), "0000000000000000000000000000000000000000")
}

func TestSortableBlobs(t *testing.T) {
	blobs := sortableBlobs{
		sortableBlob{
			nil, int64(0),
		}, sortableBlob{
			nil, int64(1),
		},
	}
	assert.True(t, blobs.Less(0, 1))
	assert.False(t, blobs.Less(1, 0))
	assert.False(t, blobs.Less(0, 0))
	blobs.Swap(0, 1)
	assert.Equal(t, blobs[0].size, int64(1))
	assert.Equal(t, blobs[1].size, int64(0))
}

func TestRenameAnalysisFork(t *testing.T) {
	ra1 := fixtureRenameAnalysis()
	clones := ra1.Fork(1)
	assert.Len(t, clones, 1)
	ra2 := clones[0].(*RenameAnalysis)
	assert.True(t, ra1 == ra2)
	ra1.Merge([]core.PipelineItem{ra2})
}

func TestRenameAnalysisSizesAreClose(t *testing.T) {
	ra := fixtureRenameAnalysis()
	assert.True(t, ra.sizesAreClose(941, 963))
	assert.True(t, ra.sizesAreClose(941, 1150))
	assert.True(t, ra.sizesAreClose(941, 803))
	assert.False(t, ra.sizesAreClose(1320, 1668))
}

func TestRenameAnalysisSortRenameCandidates(t *testing.T) {
	candidates := []int{0, 1, 2, 3}
	sortRenameCandidates(candidates, "test_regression.py", func(i int) string {
		return []string{"gather_nd_op.h", "test.py", "test_file_system.cc", "regression.py"}[i]
	})
	assert.Equal(t, candidates[0], 3)
	assert.Equal(t, candidates[1], 1)
}

func TestBlobsAreCloseText(t *testing.T) {
	blob1 := &CachedBlob{Data: []byte("hello, world!")}
	blob2 := &CachedBlob{Data: []byte("hello, world?")}
	blob1.Size = int64(len(blob1.Data))
	blob2.Size = int64(len(blob2.Data))
	ra := fixtureRenameAnalysis()
	result, err := ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.True(t, result)

	blob1.Data = []byte("hello, mloncode")
	blob1.Size = int64(len(blob1.Data))
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.False(t, result)
}

func TestBlobsAreCloseBinary(t *testing.T) {
	blob1 := &CachedBlob{}
	blob2 := &CachedBlob{}
	ra := fixtureRenameAnalysis()
	result, err := ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.True(t, result)

	blob1.Data = make([]byte, 100)
	blob2.Data = blob1.Data
	blob1.Size = 100
	blob2.Size = 100
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.True(t, result)

	blob1.Data = bytes.Repeat([]byte{1}, 100)
	blob2.Data = blob1.Data
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.True(t, result)

	for i := 0; i < 100; i++ {
		blob1.Data[i] = byte(i)
	}
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.True(t, result)

	blob2.Data = make([]byte, 100)
	for i := 0; i < 80; i++ {
		blob2.Data[i] = byte(i)
	}
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.True(t, result)

	blob2.Data[79] = 0
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.False(t, result)

	blob1.Data = []byte("hello, world!")
	blob1.Size = int64(len(blob2.Data))
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.False(t, result)

	blob1.Data, blob2.Data = blob2.Data, blob1.Data
	blob1.Size, blob2.Size = blob2.Size, blob1.Size
	result, err = ra.blobsAreClose(blob1, blob2)
	assert.Nil(t, err)
	assert.False(t, result)
}
