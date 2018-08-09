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

func TestCheckoutAndGetFile(t *testing.T) {
	client := fake.NewSimpleClientset()
	rs := NewRepoStore(client)

	repo, err := rs.Get(&RepoRef{
		URL: "https://github.com/git-fixtures/basic",
	})
	assert.Equal(t, nil, err, "Should be able to clone repo without error")

	// Check out the first commit from the REPO
	err = repo.Checkout("b029517f6300c2da0f4b651b8642506cd6aaf45d")
	assert.Equal(t, nil, err, "Should be able to checkout commit ref without error")

	// Read the LICENSE file fro the first commit
	license, err := repo.GetFile("LICENSE")
	assert.Equal(t, nil, err, "Should be able to read LICENSE file without error")
	assert.NotEqual(t, nil, license, "LICENSE file should be non-empty")

	// Try to read CHANGLOG (which doesn't exist)
	changelog, err := repo.GetFile("CHANGELOG")
	assert.NotEqual(t, nil, err, "Should not be able to read CHANGELOG, file does not exist")
	assert.NotEqual(t, nil, changelog.Contents, "CHANGELOG file should be empty")

	// Check out the second commit from the REPO
	err = repo.Checkout("b8e471f58bcbca63b07bda20e428190409c2db47")
	assert.Equal(t, nil, err, "Should be able to checkout commit ref without error")

	// Try to read CHANGLOG (which does now exist)
	// Read the CHANGELOG file from the first commit
	changelog, err = repo.GetFile("CHANGELOG")
	assert.Equal(t, nil, err, "Should be able to read CHANGELOG file without error")
	assert.Equal(t, "Initial changelog\n", changelog.Contents, "CHANGELOG file should read `Initial changelog\\n`")
}

func TestGetAllFiles(t *testing.T) {
	client := fake.NewSimpleClientset()
	rs := NewRepoStore(client)

	repo, err := rs.Get(&RepoRef{
		URL: "https://github.com/git-fixtures/basic",
	})
	assert.Equal(t, nil, err, "Should be able to clone repo without error")

	// Check out the 8th commit from the REPO
	err = repo.Checkout("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
	assert.Equal(t, nil, err, "Should be able to checkout commit ref without error")

	files, err := repo.GetAllFiles()
	assert.Equal(t, nil, err, "Should be able to read all files without error")
	assert.Equal(t, 9, len(files), "Should be 9 files in the repository")
}