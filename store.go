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
	"flag"
	"fmt"
	"sync"

	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	transportHTTP "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	transportSSH "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

var (
	insecureIgnoreHostKey = flag.Bool("inseucre-skip-host-key-verification", false, "disable host key verification for upstream SSH servers")
)

// RepoStore manages a collection of git repositories.
type RepoStore struct {
	repositories map[string]*AsyncRepoCloner
	mutex        sync.RWMutex
}

// NewRepoStore initializes a new RepoStore.
func NewRepoStore() *RepoStore {
	return &RepoStore{
		repositories: make(map[string]*AsyncRepoCloner),
		mutex:        sync.RWMutex{},
	}
}

// GetAsync returns an AsyncRepoCloner that will retrieve a Repo in the background according to the RepoRef provided.
func (rs *RepoStore) GetAsync(ref *RepoRef) (*AsyncRepoCloner, <-chan struct{}, error) {
	err := ref.Validate()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid repository reference: %v", err)
	}

	auth, err := rs.constructAuthMethod(ref)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to construct repository authentication: %v", err)
	}

	returnRC := func(rc *AsyncRepoCloner) (*AsyncRepoCloner, <-chan struct{}, error) {
		if rc.Repo != nil {
			rc.Repo.setAuth(auth)
		}

		glog.V(2).Infof("Reusing repository for %s", ref.URL)
		c := make(chan struct{})
		go func() {
			for {
				if rc.Ready {
					close(c)
					return
				}
			}
		}()
		return rc, c, nil
	}

	rs.mutex.RLock()
	if rc, ok := rs.repositories[ref.URL]; ok {
		rs.mutex.RUnlock()
		return returnRC(rc)
	}
	// Switch from read lock to write lock
	rs.mutex.RUnlock()
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	// Double check no one else has updated this in the meantime
	if rc, ok := rs.repositories[ref.URL]; ok {
		return returnRC(rc)
	}

	rc := &AsyncRepoCloner{
		RepoRef: ref,
		mutex:   sync.Mutex{},
	}

	rs.repositories[ref.URL] = rc
	done := rc.Clone(auth)
	return rc, done, nil
}

// Get retrieves a Repo from the RepoStore
func (rs *RepoStore) Get(ref *RepoRef) (*Repo, error) {
	glog.V(2).Infof("Cloning repository for %s", ref.URL)
	rc, done, err := rs.GetAsync(ref)
	if err != nil {
		return nil, err
	}

	select {
	case <-done:
		if rc.Error != nil {
			delete(rs.repositories, ref.URL)
			return nil, rc.Error
		}
		return rc.Repo, nil
	}
}

func (rs *RepoStore) constructAuthMethod(ref *RepoRef) (transport.AuthMethod, error) {
	if ref.urlType == sshURL {
		return rs.constructSSHAuthMethod(ref)
	} else if ref.urlType == httpURL {
		return rs.constructHTTPAuthMethod(ref)
	}
	return nil, nil
}

func (rs *RepoStore) constructSSHAuthMethod(ref *RepoRef) (transport.AuthMethod, error) {
	auth, err := transportSSH.NewPublicKeys(ref.User, ref.PrivateKey, ref.Pass)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	// Ignore host key validation for upstream servers
	if *insecureIgnoreHostKey {
		auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}
	return auth, nil
}

func (rs *RepoStore) constructHTTPAuthMethod(ref *RepoRef) (transport.AuthMethod, error) {
	auth := &transportHTTP.BasicAuth{
		Username: ref.User,
		Password: ref.Pass,
	}

	return auth, nil
}
