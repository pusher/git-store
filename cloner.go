package gitstore

import (
	"fmt"
	"sync"

	git "gopkg.in/src-d/go-git.v4"

	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// AsyncRepoCloner provides an asynchronous repository cloner that can perform long-running checkout operations without blocking.
type AsyncRepoCloner struct {
	Ready   bool     // Ready indicates whether the clone operation has completed.
	RepoRef *RepoRef // RepoRef is a pointer to the RepoRef handled by this cloner.
	Repo    *Repo    // Repo contains the actual repository once clone has completed.
	Error   error    // Error is the last error encountered during the clone operation or nil.
	repoDir string   // repoDir is the path to clone the repository into.
	mutex   sync.Mutex
}

// Clone starts an asynchronous clone of the requested repository and sets Ready to true when the repository is cloned successfully.
// If any errors are encountered, Ready will be false and Error will contain the error information.
func (rc *AsyncRepoCloner) Clone(auth transport.AuthMethod) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		cloneOptions := &git.CloneOptions{
			URL:  rc.RepoRef.URL,
			Auth: auth,
		}

		var err error
		var repository *git.Repository
		if rc.repoDir != "" {
			repository, err = git.PlainClone(rc.repoDir, false, cloneOptions)
		} else {
			// No repoDir provided, default to in memory clone
			fs := memfs.New()
			storer := memory.NewStorage()
			repository, err = git.Clone(storer, fs, cloneOptions)
		}

		rc.mutex.Lock()
		defer rc.mutex.Unlock()
		if err != nil {
			rc.Error = err
			return
		}
		err = cleanNewRepo(repository)
		if err != nil {
			rc.Error = fmt.Errorf("unable to clean new repo: %v", err)
			return
		}
		rc.Repo = newRepo(repository, auth)
		rc.Ready = true
	}()
	return done
}
