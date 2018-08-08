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

package store

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	transportSSH "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	insecureIgnoreHostKey = flag.Bool("inseucre-skip-host-key-verification", false, "disable host key verification for upstream SSH servers")
)

// RepoStore holds git repositories for use by the controller
type RepoStore struct {
	repositories map[string]*Repo
	client       kubernetes.Interface
}

// NewRepoStore initializes a new RepoStore
func NewRepoStore(client kubernetes.Interface) *RepoStore {
	return &RepoStore{
		repositories: make(map[string]*Repo),
		client:       client,
	}
}

// Get retrieves a Repo from the RepoStore
func (rs *RepoStore) Get(ref *RepoRef) (*Repo, error) {
	err := ref.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid reposoitory reference: %v", err)
	}

	auth, err := rs.constructAuthMethod(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to construct repository authentication: %v", err)
	}

	if r, ok := rs.repositories[ref.URL]; ok {
		r.auth = auth
		glog.V(2).Infof("Reusing repository for %s", ref.URL)
		return r, nil
	}

	glog.V(2).Infof("Cloning repository for %s", ref.URL)
	fs := memfs.New()
	storer := memory.NewStorage()
	repository, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:  ref.URL,
		Auth: auth,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to clone repository: %v", err)
	}
	repo := &Repo{
		auth:       auth,
		repository: repository,
	}

	// Store repo for reuse
	rs.repositories[ref.URL] = repo

	return repo, nil
}

func (rs *RepoStore) constructAuthMethod(ref *RepoRef) (transport.AuthMethod, error) {
	if ref.urlType == sshURL {
		return rs.constructSSHAuthMethod(ref)
	}
	return nil, nil
}

func (rs *RepoStore) constructSSHAuthMethod(ref *RepoRef) (transport.AuthMethod, error) {
	var key []byte
	if ref.SecretName != "" && ref.SecretNamespace != "" {
		secret, err := rs.client.CoreV1().Secrets(ref.SecretNamespace).Get(ref.SecretName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("unable to fetch SSH secret from kubernetes: %v", err)
		}

		var ok bool
		if key, ok = secret.Data["sshPrivateKey"]; !ok {
			return nil, fmt.Errorf("invalid secret: Secret must have key `sshPrivateKey`")
		}
	}

	auth, err := transportSSH.NewPublicKeys(ref.user, key, ref.pass)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	// Ignore host key validation for upstream servers
	if *insecureIgnoreHostKey {
		auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}
	return auth, nil
}
