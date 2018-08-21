package gitstore

import (
	git "gopkg.in/src-d/go-git.v4"

	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// AsyncRepoCloner represents an asynchronous repository cloner
type AsyncRepoCloner struct {
	Ready   bool
	RepoRef *RepoRef
	Repo    *Repo
	Error   error
}

// Clone starts an asynchronous clone of the requested repository and sets
// Ready to true when the repository is cloned successfully
func (rc *AsyncRepoCloner) Clone(auth transport.AuthMethod) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fs := memfs.New()
		storer := memory.NewStorage()
		repository, err := git.Clone(storer, fs, &git.CloneOptions{
			URL:  rc.RepoRef.URL,
			Auth: auth,
		})
		if err != nil {
			rc.Error = err
		} else {
			repo := &Repo{
				auth:       auth,
				repository: repository,
			}
			rc.Repo = repo
			rc.Ready = true
		}
	}()
	return done
}
