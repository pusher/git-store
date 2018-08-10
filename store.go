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
	"time"

	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	transportSSH "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	insecureIgnoreHostKey = flag.Bool("inseucre-skip-host-key-verification", false, "disable host key verification for upstream SSH servers")
)

// RepoStore holds git repositories for use by the controller
type RepoStore struct {
	repositories map[string]*RepoCloner
	client       kubernetes.Interface
}

// NewRepoStore initializes a new RepoStore
func NewRepoStore(client kubernetes.Interface) *RepoStore {
	return &RepoStore{
		repositories: make(map[string]*RepoCloner),
		client:       client,
	}
}

// GetAsync returns a RepoCloner that will retrieve a Repo in the background
func (rs *RepoStore) GetAsync(ref *RepoRef) (*RepoCloner, error) {
	err := ref.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid reposoitory reference: %v", err)
	}

	auth, err := rs.constructAuthMethod(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to construct repository authentication: %v", err)
	}

	var rc *RepoCloner
        if rc, ok := rs.repositories[ref.URL]; ok {
                rc.Repo.auth = auth
                glog.V(2).Infof("Reusing repository for %s", ref.URL)
		return rc, nil
        }
	rc = &RepoCloner{
		RepoRef: ref,
	};
	rs.repositories[ref.URL] = rc
	rc.Clone(auth);
	return rc, nil
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

	var rc *RepoCloner
	if rcx, ok := rs.repositories[ref.URL]; ok {
		glog.V(2).Infof("Reusing repository for %s", ref.URL)
		rcx.Repo.auth = auth
		rc = rcx
	} else {
		glog.V(2).Infof("Cloning repository for %s", ref.URL)
		rc, err = rs.GetAsync(ref)
		if (err != nil) {
			return nil, err
		}
		rs.repositories[ref.URL] = rc
	}
	for (!rc.Ready) {
		// The RepoCloner encountered an error. Return that error and remove it from store so that a fresh attempt can be made
		if (rc.Error != nil) {
			return nil, rc.Error
			delete(rs.repositories, ref.URL)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return rc.Repo, nil
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

	if ref.PrivateKey != nil {
		key = ref.PrivateKey
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
