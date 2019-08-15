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
	"context"
	"fmt"
	"sync"
	"time"
	"strings"

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
	file       *object.File
	headCommit *object.Commit
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

// cleanNewRepo ensures that the default branch of a repository is removed
// after the repository has been cloned.
// Without cleaning, a repo will always resolve the local branch rather than the
// remote branch.
func cleanNewRepo(repo *git.Repository) error {
	err := checkoutHeadHash(repo)
	if err != nil {
		return fmt.Errorf("error checking out HEAD: %v", err)
	}

	err = cleanLocalBranches(repo)
	if err != nil {
		return fmt.Errorf("error cleaning local branches: %v", err)
	}

	err = cleanLocalReferences(repo)
	if err != nil {
		return fmt.Errorf("error cleaning local references: %v", err)
	}
	return nil
}

// checkoutHeadHash detaches the worktree at the HEAD commit
// (ie no longer on a branch).
func checkoutHeadHash(repo *git.Repository) error {
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("unable to resolve HEAD commit: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to load worktree: %v", err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Hash:  head.Hash(),
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("unable to checkout HEAD hash: %v", err)
	}
	return nil
}

// cleanLocalBranches removes references to local branches from the repository
func cleanLocalBranches(repo *git.Repository) error {
	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("unable to load branches: %v", err)
	}
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			// This is a locally stored branch
			branch := strings.TrimLeft(ref.Name().String(), "refs/heads/")
			err := repo.DeleteBranch(branch)
			if err != nil {
				return fmt.Errorf("error deleting branch %s: %v", ref.Name(), err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// cleanLocalReferences removes local references from the repository
func cleanLocalReferences(repo *git.Repository) error {
	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("unable to load branches: %v", err)
	}
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			// This is a local reference and must be removed
			err := repo.Storer.RemoveReference(ref.Name())
			if err != nil {
				return fmt.Errorf("error deleting reference %s: %v", ref.Name(), err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
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
	return r.CheckoutContext(context.Background(), ref)
}

// CheckoutContext performs a Git checkout of the repository at the provided reference.
//
// Note: It is assumed that the repository has already been cloned prior to Checkout() being called.
func (r *Repo) CheckoutContext(ctx context.Context, ref string) error {
	err := r.FetchContext(ctx)
	if err != nil {
		return fmt.Errorf("unable to fetch repository: %v", err)
	}

	// Fetch the worktree
	workTree, err := r.repository.Worktree()
	if err != nil {
		return fmt.Errorf("unable to fetch repository worktree: %v", err)
	}

	hash, err := r.parseReference(ref)
	if err != nil {
		return fmt.Errorf("unable to parse ref %s: %v", ref, err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	// Perform checkout operation on worktree
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
	return r.FetchContext(context.Background())
}

// FetchContext performs a Git fetch of the repository.
//
// Note: While Fetch itself is thread-safe in that it ensures a previous Fetch() is completed before starting a new one,
// the Repo is not. If Fetch is called from two go routines, subsequent reads may be non-deterministic.
func (r *Repo) FetchContext(ctx context.Context) error {
	r.mutex.Lock()
	// Perform a fetch on the repository
	err := r.repository.FetchContext(ctx, &git.FetchOptions{
		Auth:  r.auth,
		Force: true,
		Tags:  git.AllTags,
	})
	r.mutex.Unlock()
	// Ignore "already-up-to-date" error
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("unable to fetch repository: %v", err)
	}
	return nil
}

// GetFile returns a pointer to a File from the repository that can be used to read its contents.
func (r *Repo) GetFile(path string) (*File, error) {
	commit, err := r.getHeadCommit()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD commit: %v", err)
	}

	file, err := commit.File(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load file: %v", err)
	}

	return &File{
		file:       file,
		headCommit: commit,
	}, nil
}

// GetAllFiles returns a map of Files.
// Each file is keyed in the map by it's path within the repository
func (r *Repo) GetAllFiles(subPath string, ignoreSymlinks bool) (map[string]*File, error) {
	allFiles, err := r.getAllFiles()
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
	for path, file := range allFiles {
		// If subPath is set, skip the file if it doesn't match
		if g != nil && !g.Match(path) {
			continue
		}

		// If the file is a symlink, skip it
		if ignoreSymlinks && file.file.Mode == filemode.Symlink {
			continue
		}

		files[path] = file
	}
	return files, nil
}

func (r *Repo) getAllFiles() (map[string]*File, error) {
	commit, err := r.getHeadCommit()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD commit: %v", err)
	}

	fileiter, err := commit.Files()
	if err != nil {
		return nil, fmt.Errorf("unable to load files: %v", err)
	}

	files := make(map[string]*File)
	fileiter.ForEach(func(file *object.File) error {
		files[file.Name] = &File{
			file:       file,
			headCommit: commit,
		}
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

// IsDirectory checks if the reference at a path if a directory
func (r *Repo) IsDirectory(path string) (bool, error) {
	return r.isFileMode(path, filemode.Dir)
}

// IsFile checks if the reference at a path if a regular file
func (r *Repo) IsFile(path string) (bool, error) {
	return r.isFileMode(path, filemode.Regular)
}

func (r *Repo) isFileMode(path string, mode filemode.FileMode) (bool, error) {
	commit, err := r.getHeadCommit()
	if err != nil {
		return false, fmt.Errorf("error fetching head commit: %v", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return false, fmt.Errorf("error fetching commit tree: %v", err)
	}

	entry, err := tree.FindEntry(path)
	if err != nil {
		return false, fmt.Errorf("error looking up path: %v", err)
	}
	return entry.Mode == mode, nil
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

func (f *File) getBlame() (*git.BlameResult, error) {
	blame, err := git.Blame(f.headCommit, f.file.Name)
	if err != nil {
		fmt.Printf("WARN: failed to fetch git blame: %v", err)
		return &git.BlameResult{Lines: []*git.Line{}}, nil
	}
	return blame, nil
}

// FileLog returns the file log for the current file.
func (f *File) FileLog() (GitLog, error) {
	blame, err := f.getBlame()
	if err != nil {
		return GitLog{}, fmt.Errorf("unable to get blame for %s: %v", f.file.Name, err)
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
