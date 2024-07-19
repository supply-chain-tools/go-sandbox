package gitkit

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Searcher[T SearchResult] interface {
	Process(hash *plumbing.Hash, loadData func() []byte, path string) []T
}

type RepoResult[T SearchResult] struct {
	Path            string
	Results         []T
	Matches         map[T][]BranchResult
	DanglingCommits map[T]*object.Commit
	Tags            map[T][]*object.Tag
}

type TreeSearchResult[T SearchResult] struct {
	SearchResult T
	Path         string
}

type WrappedSearchResult[T SearchResult] struct {
	SearchResult *TreeSearchResult[T]
	firstCommits []*object.Commit
	lastCommit   *object.Commit
	tags         []*object.Tag
}

type SearchResult interface {
	comparable
	MatchId() string
	SearchTermId() string
}

type BranchResult struct {
	name        string
	firstCommit []*object.Commit
	lastCommit  *object.Commit
	onTip       bool
}

func (br *BranchResult) Name() string {
	return br.name
}

func (br *BranchResult) FirstCommit() []*object.Commit {
	return br.firstCommit
}

func (br *BranchResult) LastCommit() *object.Commit {
	return br.lastCommit
}

func (br *BranchResult) OnTip() bool {
	return br.onTip
}

type TreePath struct {
	TreeHash *plumbing.Hash
	Path     string
}

type Stats struct {
	listFilesTime    int64
	queryTime        int64
	dataLoadTime     int64
	size             uint64
	numFiles         uint64
	numberOfBranches uint64
}

func (s *Stats) ListFilesTime() int64 {
	return s.listFilesTime
}

func (s *Stats) QueryTime() int64 {
	return s.queryTime
}

func (s *Stats) DataLoadTime() int64 {
	return s.dataLoadTime
}

func (s *Stats) Size() uint64 {
	return s.size
}

func (s *Stats) NumFiles() uint64 {
	return s.numFiles
}

func (s *Stats) NumberOfBranches() uint64 {
	return s.numFiles
}

func ProcessAllCommits[T SearchResult](path string,
	repo *git.Repository,
	searcher Searcher[T],
	mode Mode) ([]RepoResult[T], Stats, error) {

	start := time.Now()
	repoState := LoadRepoState(repo)
	listFilesTime := time.Since(start).Nanoseconds()
	var numberOfBranches uint64 = 0

	remotes, err := ListRemoteReferences(repo)
	if err != nil {
		return nil, Stats{}, err
	}

	processedState := NewProcessedState[T]()

	for _, remote := range remotes {
		if remote.Name() == "refs/remotes/origin/HEAD" {
			continue
		}

		numberOfBranches++
		branchName := remote.Name().String()
		commitHash := remote.Hash()

		if mode == ModeAllHistory {
			processCommitRecursively(path, &commitHash, repoState, processedState, searcher, branchName, true)
		} else {
			processCommitRecursively(path, &commitHash, repoState, processedState, searcher, branchName, false)
		}

		commitResult, found := processedState.commitResults[commitHash]
		if found {
			result := make(map[*TreeSearchResult[T]]*WrappedSearchResult[T])
			for _, sr := range commitResult {
				result[sr.SearchResult] = sr
			}

			processedState.branchResults[branchName] = result
		} else {
			log.Fatalf("Failed to get result for commit %s, branch '%s'", commitHash, branchName)
		}
	}

	if mode == ModeAllHistory {
		danglingCommits := make([]*plumbing.Hash, 0)
		for commitHash := range repoState.commitMap {
			_, found := processedState.commitResults[commitHash]
			if !found {
				danglingCommits = append(danglingCommits, &commitHash)
			}
		}

		danglingCommitResults := make(map[*TreeSearchResult[T]]*object.Commit)
		for _, commitHash := range danglingCommits {
			processCommitRecursively(path, commitHash, repoState, processedState, searcher, "", true)
			commitResult, found := processedState.commitResults[*commitHash]
			if found {
				for _, sr := range commitResult {
					if len(sr.firstCommits) == 1 && sr.firstCommits[0].Hash.String() == commitHash.String() {
						danglingCommitResults[sr.SearchResult] = sr.firstCommits[0]
					}
				}
			} else {
				log.Fatalf("Failed to get result for commit %s", *commitHash)
			}
		}
		// TODO dangling object
	}

	repoResults := make([]RepoResult[T], 0)

	for path, matchMap := range processedState.results {
		branchResults := make(map[T][]BranchResult)
		danglingResults := make(map[T]*object.Commit)
		tagResults := make(map[T][]*object.Tag)
		results := make([]T, 0)
		for _, resultMap := range matchMap {
			for _, result := range resultMap {
				br := make([]BranchResult, 0)
				for _, remote := range remotes {
					if remote.Name() == "refs/remotes/origin/HEAD" {
						continue
					}

					head := remote.Hash().String()

					branchName := remote.Name().String()
					r, found := processedState.branchResults[branchName]

					if found {
						resultWrapper, found := r[result]
						if found {
							br = append(br, BranchResult{
								name:        branchName,
								firstCommit: resultWrapper.firstCommits,
								lastCommit:  resultWrapper.lastCommit,
								onTip:       resultWrapper.lastCommit.Hash.String() == head,
							})
						}
					}
				}

				results = append(results, result.SearchResult)
				branchResults[result.SearchResult] = br

				dangling, haveDanglingCommits := danglingResults[result.SearchResult]
				if haveDanglingCommits {
					danglingResults[result.SearchResult] = dangling
				}

				tags, haveTags := processedState.tagResults[result]
				if haveTags {
					tagResults[result.SearchResult] = tags
				}
			}
		}

		repoResults = append(repoResults, RepoResult[T]{
			Results:         results,
			Path:            path,
			Matches:         branchResults,
			DanglingCommits: danglingResults,
			Tags:            tagResults,
		})
	}

	return repoResults, Stats{
		queryTime:        processedState.queryTime,
		dataLoadTime:     processedState.dataLoadTime,
		listFilesTime:    listFilesTime,
		numFiles:         processedState.numFiles,
		size:             processedState.size,
		numberOfBranches: numberOfBranches,
	}, nil
}

func processCommitRecursively[T SearchResult](path string, commitHash *plumbing.Hash,
	repoState *RepoState,
	processedState *ProcessedState[T],
	searcher Searcher[T],
	branchName string,
	recurse bool) {
	hash := *commitHash

	_, alreadyProcessed := processedState.commitResults[hash]
	if alreadyProcessed {
		return
	}

	commit, found := repoState.commitMap[hash]
	if !found {
		log.Fatalf("did not find commit '%s' for repo '%s'", hash, path)
	}

	if recurse {
		for _, parent := range commit.ParentHashes {
			processCommitRecursively(path, &parent, repoState, processedState, searcher, branchName, recurse)
		}
	}

	processCommit(commit, repoState, processedState, searcher, recurse)

	return
}

func processCommit[T SearchResult](commit *object.Commit,
	repoState *RepoState,
	processedState *ProcessedState[T],
	searcher Searcher[T],
	recurse bool) {

	results := processTreeRecursively(commit, repoState, processedState, searcher, commit.TreeHash, "")

	commitResultsMap := make(map[*TreeSearchResult[T]]*WrappedSearchResult[T])

	if recurse {
		for _, parentHash := range commit.ParentHashes {
			parentResults := processedState.commitResults[parentHash]

			for _, parentResult := range parentResults {
				existingElement, found := commitResultsMap[parentResult.SearchResult]
				if found {
					allFirstCommitsSet := hashset.New[*object.Commit]()

					for _, firstCommit := range existingElement.firstCommits {
						allFirstCommitsSet.Add(firstCommit)
					}

					for _, firstCommit := range parentResult.firstCommits {
						allFirstCommitsSet.Add(firstCommit)
					}

					allFirstCommit := make([]*object.Commit, 0)
					for _, j := range allFirstCommitsSet.Values() {
						allFirstCommit = append(allFirstCommit, j)
					}

					existingElement.firstCommits = allFirstCommit
				} else {
					commitResultsMap[parentResult.SearchResult] = &WrappedSearchResult[T]{
						firstCommits: parentResult.firstCommits,
						lastCommit:   parentResult.lastCommit,
						SearchResult: parentResult.SearchResult,
					}
				}
			}
		}
	}

	for _, r := range results {
		element, found := commitResultsMap[r]
		if found {
			element.lastCommit = commit
		} else {
			element = &WrappedSearchResult[T]{
				firstCommits: []*object.Commit{commit},
				lastCommit:   commit,
				SearchResult: r,
			}
			commitResultsMap[r] = element
		}

		tags, tagsForCommit := repoState.targetToTagMap[commit.Hash]
		if tagsForCommit {
			_, haveTagResults := processedState.tagResults[r]
			if !haveTagResults {
				processedState.tagResults[r] = make([]*object.Tag, 0)
			}
			processedState.tagResults[r] = append(processedState.tagResults[r], tags...)
		}
	}

	commitResults := make([]*WrappedSearchResult[T], 0)
	for _, v := range commitResultsMap {
		commitResults = append(commitResults, v)
	}

	processedState.commitResults[commit.Hash] = commitResults
}

func processTreeRecursively[T SearchResult](commit *object.Commit,
	repoState *RepoState,
	processedState *ProcessedState[T],
	searcher Searcher[T],
	treeHash plumbing.Hash,
	path string) []*TreeSearchResult[T] {

	existingTreeResults, found := processedState.treeResults[treeHash]
	if found {
		return existingTreeResults
	}

	tree, found := repoState.treeMap[treeHash]
	if !found {
		log.Fatalf("did not find tree hash '%s'", treeHash)
	}

	treeResults := hashset.New[*TreeSearchResult[T]]()
	for _, entry := range tree.Entries {
		if entry.Mode == filemode.Dir {
			recursiveResults := processTreeRecursively(commit, repoState, processedState, searcher, entry.Hash, path+entry.Name+"/")
			for _, recursiveResult := range recursiveResults {
				treeResults.Add(recursiveResult)
			}
		}
	}

	for _, entry := range tree.Entries {
		if entry.Mode == filemode.Regular || entry.Mode == filemode.Executable || entry.Mode == filemode.Deprecated {
			currentPath := path + entry.Name

			var dataTime int64 = 0
			dataLoader := func() []byte {
				blob, blobFound := repoState.blobMap[entry.Hash]
				if !blobFound {
					log.Fatalf("could not find blob '%s'\n", entry.Hash)
				}

				start := time.Now()
				r, err := blob.Reader()
				if err != nil {
					log.Fatal(err)
				}

				bytes, err := io.ReadAll(r)
				if err != nil {
					log.Fatal(err)
				}
				dataTime = time.Since(start).Nanoseconds()
				processedState.dataLoadTime += dataTime

				processedState.numFiles += 1
				processedState.size += uint64(len(bytes))

				return bytes
			}

			start := time.Now()
			results := searcher.Process(&entry.Hash, dataLoader, currentPath)
			elapsed := time.Since(start).Nanoseconds()
			processedState.queryTime += elapsed - dataTime

			pathMap, ok := processedState.results[currentPath]
			if !ok {
				pathMap = make(map[string]map[string]*TreeSearchResult[T])
				processedState.results[currentPath] = pathMap
			}

			for _, result := range results {
				matchMap, ok := pathMap[result.MatchId()]
				if !ok {
					matchMap = make(map[string]*TreeSearchResult[T])
					pathMap[result.MatchId()] = matchMap
				}

				existingResult, found := matchMap[result.SearchTermId()]
				if !found {
					newResult := &TreeSearchResult[T]{
						SearchResult: result,
						Path:         currentPath,
					}
					matchMap[result.SearchTermId()] = newResult
					treeResults.Add(newResult)
				} else {
					treeResults.Add(existingResult)
				}
			}
		}
	}

	results := make([]*TreeSearchResult[T], 0)
	for _, treeResult := range treeResults.Values() {
		results = append(results, treeResult)
	}

	processedState.treeResults[treeHash] = results

	return results
}

func LoadRepoState(repo *git.Repository) *RepoState {
	iter, err := repo.Storer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		log.Fatal(err)
	}

	repoState := newRepoState()

	processedObject := hashset.New[plumbing.Hash]()
	err = iter.ForEach(func(obj plumbing.EncodedObject) error {
		if processedObject.Contains(obj.Hash()) {
			slog.Debug("skipping object", "type", obj.Type(), "hash", obj.Hash().String())
			return nil
		}

		obj.Hash()
		switch obj.Type() {
		case plumbing.BlobObject:
			// process and store result rather than data
			blob := &object.Blob{}
			err := blob.Decode(obj)
			if err != nil {
				log.Fatal(err)
			}
			repoState.blobMap[obj.Hash()] = blob
		case plumbing.TreeObject:
			tree := &object.Tree{}
			err := tree.Decode(obj)
			if err != nil {
				log.Fatal(err)
			}
			repoState.treeMap[obj.Hash()] = tree
		case plumbing.CommitObject:
			commit := &object.Commit{}
			err := commit.Decode(obj)
			if err != nil {
				log.Fatal(err)
			}
			repoState.commitMap[obj.Hash()] = commit
		case plumbing.TagObject:
			tag := &object.Tag{}
			err := tag.Decode(obj)
			if err != nil {
				log.Fatal(err)
			}
			repoState.tagMap[obj.Hash()] = tag
		default:
			log.Fatal("unknown object type")
		}

		slog.Debug("Processed object", "type", obj.Type(), "hash", obj.Hash())
		processedObject.Add(obj.Hash())

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range repoState.tagMap {
		existing, found := repoState.targetToTagMap[v.Target]
		if found {
			repoState.targetToTagMap[v.Target] = append(existing, v)
		} else {
			repoState.targetToTagMap[v.Target] = []*object.Tag{v}
		}
	}

	return repoState
}

type RepoState struct {
	blobMap        map[plumbing.Hash]*object.Blob
	treeMap        map[plumbing.Hash]*object.Tree
	commitMap      map[plumbing.Hash]*object.Commit
	tagMap         map[plumbing.Hash]*object.Tag
	targetToTagMap map[plumbing.Hash][]*object.Tag
}

func (rs *RepoState) GetCommitMap() map[plumbing.Hash]*object.Commit {
	return rs.commitMap
}

func newRepoState() *RepoState {
	blobMap := make(map[plumbing.Hash]*object.Blob)
	treeMap := make(map[plumbing.Hash]*object.Tree)
	commitMap := make(map[plumbing.Hash]*object.Commit)
	tagMap := make(map[plumbing.Hash]*object.Tag)
	targetToTagMap := make(map[plumbing.Hash][]*object.Tag)

	return &RepoState{
		blobMap:        blobMap,
		treeMap:        treeMap,
		commitMap:      commitMap,
		tagMap:         tagMap,
		targetToTagMap: targetToTagMap,
	}
}

type ProcessedState[T SearchResult] struct {
	// path, match, searchTerm
	results map[string]map[string]map[string]*TreeSearchResult[T]

	treeResults map[plumbing.Hash][]*TreeSearchResult[T]

	// branch name
	branchResults map[string]map[*TreeSearchResult[T]]*WrappedSearchResult[T]
	commitResults map[plumbing.Hash][]*WrappedSearchResult[T]
	tagResults    map[*TreeSearchResult[T]][]*object.Tag
	size          uint64
	numFiles      uint64
	dataLoadTime  int64
	queryTime     int64
}

func NewProcessedState[T SearchResult]() *ProcessedState[T] {
	results := make(map[string]map[string]map[string]*TreeSearchResult[T])
	treeResults := make(map[plumbing.Hash][]*TreeSearchResult[T])
	branchResults := make(map[string]map[*TreeSearchResult[T]]*WrappedSearchResult[T])
	commitResults := make(map[plumbing.Hash][]*WrappedSearchResult[T])
	tagResults := make(map[*TreeSearchResult[T]][]*object.Tag)

	return &ProcessedState[T]{
		results:       results,
		treeResults:   treeResults,
		branchResults: branchResults,
		commitResults: commitResults,
		tagResults:    tagResults,
		size:          0,
		numFiles:      0,
		dataLoadTime:  0,
		queryTime:     0,
	}
}

func ProcessAllFiles[T SearchResult](searcher Searcher[T], repo Repository) ([]RepoResult[T], Stats, error) {
	repoPath := repo.LocalRootPath()
	if !strings.HasSuffix(repoPath, "/") {
		repoPath += "/"
	}

	searchPath := repoPath
	if repo.SubPath() != nil {
		searchPath += *repo.SubPath()
	}

	start := time.Now()
	files, err := listAllFilesInLocalPath(searchPath)
	if err != nil {
		return nil, Stats{}, err
	}

	listFilesTime := time.Since(start).Nanoseconds()

	result := make([]RepoResult[T], 0)

	var localQueryTime int64 = 0
	var localDataLoadTime int64 = 0
	var totalSize uint64
	var numberOfFiles uint64

	for _, file := range files {
		localPath := file[len(repoPath):]

		var dataTime int64 = 0
		loadData := func() []byte {
			start := time.Now()
			data, err := os.ReadFile(file)
			if err != nil {
				log.Fatal(err)
			}
			dataTime = time.Since(start).Nanoseconds()

			localDataLoadTime += dataTime
			totalSize += uint64(len(data))
			numberOfFiles += 1

			return data
		}

		start = time.Now()
		r := searcher.Process(nil, loadData, localPath)
		elapsed := time.Since(start).Nanoseconds()
		localQueryTime += elapsed - dataTime

		if len(r) > 0 {
			result = append(result, RepoResult[T]{
				Path:    localPath,
				Results: r,
			})
		}
	}

	return result, Stats{
		queryTime:        localQueryTime,
		dataLoadTime:     localDataLoadTime,
		listFilesTime:    listFilesTime,
		numFiles:         numberOfFiles,
		size:             totalSize,
		numberOfBranches: 0,
	}, nil
}

func listAllFilesInLocalPath(path string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if !info.IsDir() && info.Type() != os.ModeSymlink {
			paths = append(paths, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get all local files from path '%s': %w", path, err)
	}

	return paths, nil
}
