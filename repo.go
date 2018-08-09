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
	"time"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

// Repo represents a git repository
type Repo struct {
	auth       transport.AuthMethod
	repository *git.Repository
}

// File represents a file content and log pair from the repository
type File struct {
	file *object.File
	Log  GitLog
}

// GitLog contains information about a commit from the git repository log
type GitLog struct {
	Date   time.Time
	Hash   plumbing.Hash
	Author string
	Text   string
}

// Checkout performs a Git checkout of the repository at the given reference
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

	var hash *plumbing.Hash
	hash, err = r.repository.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		hash, err = r.repository.ResolveRevision(plumbing.Revision(fmt.Sprintf("%s/%s", "refs/remotes/origin", ref)))
		if err != nil {
			return fmt.Errorf("unable to parse ref %s: %v", ref, err)
		}
	}

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

// Fetch performs a Git fetch of the repository
func (r *Repo) Fetch() error {
	// Perform a fetch on the repository
	err := r.repository.Fetch(&git.FetchOptions{
		Auth:  r.auth,
		Force: true,
		Tags:  git.AllTags,
	})
	// Ignore "already-up-to-date" error
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("unable to fetch repository: %v", err)
	}
	return nil
}

// GetFile returns the contents of a file from within the repository
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
	head, err := r.repository.Head()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD: %v", err)
	}

	commit, err := r.repository.CommitObject(head.Hash())
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
	head, err := r.repository.Head()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD: %v", err)
	}

	commit, err := r.repository.CommitObject(head.Hash())
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
func (r *Repo) GetAllFiles() (map[string]*File, error) {
	rawFiles, err := r.getAllFiles()
	if err != nil {
		return nil, fmt.Errorf("unable to read files from repository: %v", err)
	}

	files := make(map[string]*File)
	for path, file := range rawFiles {
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
	head, err := r.repository.Head()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch HEAD: %v", err)
	}

	commit, err := r.repository.CommitObject(head.Hash())
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

// Contents returns the content of a file
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
