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
	"testing"

	"github.com/bmizerany/assert"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGet(t *testing.T) {
	client := fake.NewSimpleClientset()
	rs := NewRepoStore(client)

	// Clone a test repo
	repo, err := rs.Get(&RepoRef{
		URL: "https://github.com/git-fixtures/basic",
	})
	assert.Equal(t, nil, err, "Should be able to clone repo without error")

	// Check the origin remote was set correctly
	origin, err := repo.repository.Remote("origin")
	assert.Equal(t, nil, err, "Should be able to get origin remote without error")
	assert.Equal(t, "origin", origin.Config().Name, "origin remote name not as expected")
	assert.Equal(t, []string{"https://github.com/git-fixtures/basic"}, origin.Config().URLs, "origin remote URLs not as expected")

	// Check out the first commit from the REPO
	err = repo.Checkout("b029517f6300c2da0f4b651b8642506cd6aaf45d")
	assert.Equal(t, nil, err, "Should be able to checkout commit ref without error")

	// Check out the master branch from the REPO
	err = repo.Checkout("master")
	assert.Equal(t, nil, err, "Should be able to checkout commit ref without error")
}
