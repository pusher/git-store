package gitstore

import (
        git "gopkg.in/src-d/go-git.v4"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
        "gopkg.in/src-d/go-billy.v4/memfs"
        "gopkg.in/src-d/go-git.v4/storage/memory"
)

type RepoCloner struct {
	Ready		bool
	RepoRef		*RepoRef
	Repo		*Repo
	Error		error
}


func (rc *RepoCloner) Clone(rs *RepoStore, auth transport.AuthMethod) {
	go func() {
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
			rs.repositories[rc.RepoRef.URL] = repo
			rc.Repo = repo
			rc.Ready = true			
		}
	}();
}
