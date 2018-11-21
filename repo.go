/*
Copyright 2018 Pusher Ltd.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gitstore

import (
	"fmt"
	"sync"
	"time"

	"github.com/gobwas/glob"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

// Repo represents a git repository.
type Repo struct {
	auth       transport.AuthMethod
	repository *git.Repository
	mutex      sync.RWMutex
}

// File represents a file within a git repository.
type File struct {
	Log  GitLog // Log contians the git log information for this file at the current reference.
	file *object.File
}

// GitLog contains information about a commit from the git repository log.
type GitLog struct {
	Date   time.Time     // Date is the datetime of the commit this log corresponds to.
	Hash   plumbing.Hash // Hash contains the hash of the commit.
	Author string        // Author is the author as stored in the commit.
	Text   string        // Text is the commit message.
}

// newRepo constructs a new Repo with all required fields set
func newRepo(repo *git.Repository, auth transport.AuthMethod) *Repo {
	return &Repo{
		repository: repo,
		auth:       auth,
		mutex:      sync.RWMutex{},
	}
}

// setAuth sets the repositories auth method
func (r *Repo) setAuth(auth transport.AuthMethod) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.auth = auth
}

// Checkout performs a Git checkout of the repository at the provided reference.
//
// Note: It is assumed that the repository has already been cloned prior to Checkout() being called.
func (r *Repo) Checkout(ref string) error {
	err := r.Fetch()
	if err != nil {
		return fmt.Errorf("unable to fetch repository: %v", err)
	}

	// Fetch the worktree
	workTree, err := r.repository.Worktree()
	if err != nil {
		return fmt.Errorf("unable to fetch repository worktree: %v", err)
	}

	// Resolve remote master not local branch
	if ref == "master" {
		ref = "refs/remotes/origin/master"
	}

	hash, err := r.parseReference(ref)
	if err != nil {
		return fmt.Errorf("unable to parse ref %s: %v", ref, err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	//Perform checkout operation on worktree
	err = workTree.Checkout(&git.CheckoutOptions{
		Hash:  *hash,
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("unable to checkout reference %s: %v", ref, err)
	}
	return nil
}

// parseReference attempts to convert the git reference into a hash
func (r *Repo) parseReference(ref string) (*plumbing.Hash, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	// attempt to parse ref as it is
	hash, err := r.repository.ResolveRevision(plumbing.Revision(ref))
	if err == nil {
		// No error so return hash
		return hash, nil
	}
	// attempt to pars ref prefixed by 'refs/remotes/origin'
	hash, err = r.repository.ResolveRevision(plumbing.Revision(fmt.Sprintf("%s/%s", "refs/remotes/origin", ref)))
	if err == nil {
		// No error so return hash
		return hash, nil
	}
	return nil, err
}

// Fetch performs a Git fetch of the repository.
//
// Note: While Fetch itself is thread-safe in that it ensures a previous Fetch() is completed before starting a new one,
// the Repo is not. If Fetch is called from two go routines, subsequent reads may be non-deterministic.
func (r *Repo) Fetch() error {
	// Perform a fetch on the repository
	err := r.fetch()
	// Ignore "already-up-to-date" error
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("unable to fetch repository: %v", err)
	}
	return nil
}

// fetch performs a fetch on the internal repository while under a lock
func (r *Repo) fetch() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.repository.Fetch(&git.FetchOptions{
		Auth:  r.auth,
		Force: true,
		Tags:  git.AllTags,
	})
}

/*
GetFile returns a pointer to a File from the repository that can be used to read its contents.
*/
func (r *Repo) GetFile(path string) (*File, error) {
	// Open file from repository
	file, err := r.getFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load file %s: %v", path, err)
	}

	fileLog, err := r.getFileLog(path)
	if err != nil {
		return nil, fmt.Errorf("unable to get log: %v", err)
	}

	return &File{
		file: file,
		Log:  fileLog,
	}, nil
}

func (r *Repo) getFile(path string) (*object.File, error) {
	commit, err := r.getHeadCommit()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD commit: %v", err)
	}

	file, err := commit.File(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load file: %v", err)
	}
	return file, nil
}

func (r *Repo) getBlame(path string) (*git.BlameResult, error) {
	commit, err := r.getHeadCommit()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD commit: %v", err)
	}

	blame, err := git.Blame(commit, path)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch git blame: %v", err)
	}
	return blame, nil
}

func (r *Repo) getFileLog(path string) (GitLog, error) {
	blame, err := r.getBlame(path)
	if err != nil {
		return GitLog{}, fmt.Errorf("unable to get blame for %s: %v", path, err)
	}

	var fileLog GitLog
	for _, line := range blame.Lines {
		if line.Date.After(fileLog.Date) {
			fileLog = GitLog{
				Date:   line.Date,
				Hash:   line.Hash,
				Author: line.Author,
				Text:   line.Text,
			}
		}
	}

	return fileLog, nil
}

// GetAllFiles returns a map of Files.
// Each file is keyed in the map by it's path within the repository
func (r *Repo) GetAllFiles(subPath string, ignoreSymlinks bool) (map[string]*File, error) {
	rawFiles, err := r.getAllFiles()
	if err != nil {
		return nil, fmt.Errorf("unable to read files from repository: %v", err)
	}

	var g glob.Glob
	if subPath != "" {
		g, err = glob.Compile(subPath)
		if err != nil {
			return nil, fmt.Errorf("unable to compile subPath matcher: %v", err)
		}
	}

	files := make(map[string]*File)
	for path, file := range rawFiles {
		// If subPath is set, skip the file if it doesn't match
		if g != nil && !g.Match(path) {
			continue
		}

		// If the file is a symlink, skip it
		if ignoreSymlinks && file.Mode == filemode.Symlink {
			continue
		}

		fileLog, err := r.getFileLog(path)
		if err != nil {
			return nil, fmt.Errorf("unable to get log for %s: %v", path, err)
		}
		files[path] = &File{
			file: file,
			Log:  fileLog,
		}
	}
	return files, nil
}

func (r *Repo) getAllFiles() (map[string]*object.File, error) {
	commit, err := r.getHeadCommit()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD commit: %v", err)
	}

	fileiter, err := commit.Files()
	if err != nil {
		return nil, fmt.Errorf("unable to load files: %v", err)
	}

	files := make(map[string]*object.File)
	fileiter.ForEach(func(file *object.File) error {
		files[file.Name] = file
		return nil
	})

	return files, nil
}

// LastUpdated returns the timestamp that the currently checked out reference was last updated at.
func (r *Repo) LastUpdated() (time.Time, error) {
	commit, err := r.getHeadCommit()
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to fetch HEAD commit: %v", err)
	}

	return commit.Committer.When, nil
}

func (r *Repo) getHeadCommit() (*object.Commit, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	head, err := r.repository.Head()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch repository head: %v", err)
	}

	commit, err := r.repository.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve commit: %v", err)
	}
	return commit, nil
}

// Contents returns the content of the File as a string.
//
// Note: Contents() does not verify file type and will return binary files as a (probably useless) string representation.
// It reads the contents in memory, so may suffer from problems if the file size is too large.
func (f *File) Contents() string {
	if f.file == nil {
		return ""
	}
	content, err := f.file.Contents()
	if err != nil {
		return ""
	}
	return content
}
