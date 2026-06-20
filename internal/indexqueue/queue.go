package indexqueue

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/kernelcode0/aptify/internal/index"
	"github.com/kernelcode0/aptify/internal/storage"
)

type repoState struct {
	indexing    bool
	pending     bool
	lastIndexed *time.Time
}

// Queue is an in-process async worker for regenerating APT indexes.
// A single goroutine processes jobs; duplicate enqueues for the same repo
// while a job is running collapse into one follow-up run.
type Queue struct {
	gen    *index.Generator
	db     *storage.DB
	mu     sync.Mutex
	states map[string]*repoState
	jobs   chan string // buffered channel of repo IDs
}

// Status reports the current indexing state for a repo.
type Status struct {
	Indexing    bool       `json:"indexing"`
	LastIndexed *time.Time `json:"last_indexed"`
}

func New(gen *index.Generator, db *storage.DB) *Queue {
	return &Queue{
		gen:    gen,
		db:     db,
		states: make(map[string]*repoState),
		jobs:   make(chan string, 256),
	}
}

// Start launches the background worker. ctx cancellation stops the worker.
func (q *Queue) Start(ctx context.Context) {
	go q.worker(ctx)
}

// Enqueue schedules an index regeneration for repoID. If a job is already
// running for that repo, a follow-up run is scheduled instead (deduplicated).
func (q *Queue) Enqueue(repoID string) {
	q.mu.Lock()
	state, ok := q.states[repoID]
	if !ok {
		state = &repoState{}
		q.states[repoID] = state
	}
	if state.indexing {
		state.pending = true
		q.mu.Unlock()
		return
	}
	state.indexing = true
	q.mu.Unlock()

	q.jobs <- repoID
}

// RepoStatus returns the current indexing state for repoID.
func (q *Queue) RepoStatus(repoID string) Status {
	q.mu.Lock()
	defer q.mu.Unlock()
	state, ok := q.states[repoID]
	if !ok {
		return Status{}
	}
	return Status{
		Indexing:    state.indexing,
		LastIndexed: state.lastIndexed,
	}
}

func (q *Queue) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case repoID := <-q.jobs:
			q.run(repoID)
		}
	}
}

func (q *Queue) run(repoID string) {
	repo, err := q.db.GetRepo(repoID)
	if err != nil || repo == nil {
		log.Printf("indexqueue: repo %s not found: %v", repoID, err)
		q.finish(repoID, false)
		return
	}
	packages, err := q.db.ListPackages(repoID, 0, 0)
	if err != nil {
		log.Printf("indexqueue: list packages for repo %s: %v", repoID, err)
		q.finish(repoID, false)
		return
	}
	if err := q.gen.Regenerate(repo, packages); err != nil {
		log.Printf("indexqueue: regenerate index for repo %s: %v", repoID, err)
		q.finish(repoID, false)
		return
	}
	q.finish(repoID, true)
}

func (q *Queue) finish(repoID string, success bool) {
	q.mu.Lock()
	state, ok := q.states[repoID]
	if !ok {
		q.mu.Unlock()
		return
	}
	if success {
		now := time.Now().UTC()
		state.lastIndexed = &now
	}
	requeue := state.pending
	state.pending = false
	if !requeue {
		state.indexing = false
	}
	q.mu.Unlock()

	if requeue {
		// Worker is idle now; send next job (no mutex held, no deadlock risk).
		q.jobs <- repoID
	}
}
