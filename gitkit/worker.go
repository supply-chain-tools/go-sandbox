package gitkit

import (
	"log"
	"sync"
	"sync/atomic"
)

type Mode string

const (
	ModeAllHistory  Mode = "all-history"
	ModeAllBranches Mode = "all-branches"
	ModeAllFiles    Mode = "all-files"
)

func Search[T SearchResult](repos []Repository,
	newSearcher func(Repository) Searcher[T],
	mode Mode,
	concurrency int) (chan WorkerResult[T], *WorkerStats) {

	tasksChannel := make(chan Repository, len(repos))
	for _, repo := range repos {
		tasksChannel <- repo
	}

	resultsChannel := make(chan WorkerResult[T])

	numberOfFiles := atomic.Uint64{}
	totalFileSize := atomic.Uint64{}
	queryTime := atomic.Int64{}
	dataLoadTime := atomic.Int64{}
	listFilesTime := atomic.Int64{}
	numberOfBranches := atomic.Uint64{}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		worker := worker[T]{
			taskQueue:        tasksChannel,
			resultsChannel:   resultsChannel,
			waitGroup:        waitGroup,
			numberOfFiles:    &numberOfFiles,
			numberOfBranches: &numberOfBranches,
			totalFileSize:    &totalFileSize,
			queryTimer:       &queryTime,
			dataLoadTimer:    &dataLoadTime,
			listFilesTimer:   &listFilesTime,
			mode:             mode,
			newSearcher:      newSearcher,
		}
		worker.start()
	}

	close(tasksChannel)
	go func() {
		waitGroup.Wait()
		close(resultsChannel)
	}()

	return resultsChannel, &WorkerStats{
		numberOfFiles:    &numberOfFiles,
		totalFileSize:    &totalFileSize,
		queryTime:        &queryTime,
		dataLoadTime:     &dataLoadTime,
		listFilesTime:    &listFilesTime,
		numberOfRepos:    len(repos),
		numberOfBranches: &numberOfBranches,
	}
}

type WorkerResult[T SearchResult] struct {
	Repo        Repository
	Ref         string
	RepoResults []RepoResult[T]
}

type worker[T SearchResult] struct {
	taskQueue        <-chan Repository
	resultsChannel   chan<- WorkerResult[T]
	waitGroup        *sync.WaitGroup
	totalFileSize    *atomic.Uint64
	numberOfFiles    *atomic.Uint64
	numberOfBranches *atomic.Uint64
	queryTimer       *atomic.Int64
	dataLoadTimer    *atomic.Int64
	listFilesTimer   *atomic.Int64
	mode             Mode
	newSearcher      func(Repository) Searcher[T]
}

func (w *worker[T]) start() {
	go func() {
		for task := range w.taskQueue {
			var result []RepoResult[T]
			var stats Stats

			searcher := w.newSearcher(task)
			if w.mode == ModeAllHistory || w.mode == ModeAllBranches {
				repo, err := OpenRepoInLocalPath(task.LocalRootPath())
				if err != nil {
					log.Printf("Error opening repo %s (skipping): %v", task.LocalRootPath(), err)
					continue
				}

				result, stats, err = ProcessAllCommits[T](task.LocalRootPath(), repo, searcher, w.mode)
				if err != nil {
					log.Printf("Error processing repo %s (skipping): %v", task.LocalRootPath(), err)
					continue
				}
			} else {
				var err error
				result, stats, err = ProcessAllFiles[T](searcher, task)
				if err != nil {
					log.Printf("Error processing repo %s (skipping): %v", task.LocalRootPath(), err)
					continue
				}
			}

			w.totalFileSize.Add(stats.Size())
			w.numberOfFiles.Add(stats.NumFiles())
			w.listFilesTimer.Add(stats.ListFilesTime())
			w.queryTimer.Add(stats.QueryTime())
			w.dataLoadTimer.Add(stats.DataLoadTime())
			w.numberOfBranches.Add(stats.NumberOfBranches())

			if len(result) > 0 {
				// FIXME don't open repo again or at all when searching files
				repo, err := OpenRepoInLocalPath(task.LocalRootPath())
				if err != nil {
					log.Printf("Error opening repo %s (skipping): %v", task.LocalRootPath(), err)
					continue
				}

				ref, err := repo.Head()
				if err != nil {
					log.Fatal("unable to get branch, but have results")
				}

				w.resultsChannel <- WorkerResult[T]{
					Repo:        task,
					RepoResults: result,
					Ref:         ref.Hash().String(),
				}
			}
		}
		w.waitGroup.Done()
	}()
}

type WorkerStats struct {
	numberOfFiles    *atomic.Uint64
	totalFileSize    *atomic.Uint64
	queryTime        *atomic.Int64
	dataLoadTime     *atomic.Int64
	listFilesTime    *atomic.Int64
	numberOfRepos    int
	numberOfBranches *atomic.Uint64
}

func (ws *WorkerStats) NumberOfFiles() uint64 {
	return ws.numberOfFiles.Load()
}

func (ws *WorkerStats) TotalFileSize() uint64 {
	return ws.totalFileSize.Load()
}

func (ws *WorkerStats) QueryTime() int64 {
	return ws.queryTime.Load()
}

func (ws *WorkerStats) DataLoadTime() int64 {
	return ws.dataLoadTime.Load()
}

func (ws *WorkerStats) ListFilesTime() int64 {
	return ws.listFilesTime.Load()
}

func (ws *WorkerStats) NumberOfRepos() int {
	return ws.numberOfRepos
}

func (ws *WorkerStats) NumberOfBranches() uint64 {
	return ws.numberOfBranches.Load()
}
