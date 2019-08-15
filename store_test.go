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
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GitStore", func() {

	Context("When able to clone repo without error", func() {
		var rs *RepoStore
		var repo *Repo

		BeforeEach(func() {
			rs = NewRepoStore("")
			var err error
			repo, err = rs.Get(&RepoRef{
				URL: repositoryURL,
			})
			Expect(err).ToNot(HaveOccurred())
			err = repo.Checkout("master")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should set the origin remote correctly", func() {
			origin, err := repo.repository.Remote("origin")
			Expect(err).ToNot(HaveOccurred())
			Expect(origin.Config().Name).To(Equal("origin"))
		})

		It("Should be able to checkout the first commit from the repo", func() {
			err := repo.Checkout("b029517f6300c2da0f4b651b8642506cd6aaf45d")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should be able to checkout the master branch from the repo", func() {
			err := repo.Checkout("master")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When cloning into a directory", func() {
		var rs *RepoStore
		var tmpDir string

		BeforeEach(func() {
			// Create a temp directory for each test
			var err error
			tmpDir, err = ioutil.TempDir("", "git-store")
			Expect(err).To(BeNil())
			rs = NewRepoStore(tmpDir)
		})

		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		It("can clone a repository without error", func() {
			_, err := rs.Get(&RepoRef{
				URL: repositoryURL,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("once cloned", func() {
			var repo *Repo

			BeforeEach(func() {
				var err error
				repo, err = rs.Get(&RepoRef{
					URL: repositoryURL,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			It("can checkout master", func() {
				err := repo.Checkout("master")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should clone into a directory in the tmpDir", func() {
				info, err := os.Stat(filepath.Join(tmpDir, repositoryURL))
				Expect(err).To(BeNil())
				Expect(info.IsDir()).To(BeTrue())
			})

			Context("and master is checked out", func() {
				BeforeEach(func() {
					err := repo.Checkout("master")
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should set the origin remote correctly", func() {
					origin, err := repo.repository.Remote("origin")
					Expect(err).ToNot(HaveOccurred())
					Expect(origin.Config().Name).To(Equal("origin"))
				})

				It("Should be able to checkout the first commit from the repo", func() {
					err := repo.Checkout("b029517f6300c2da0f4b651b8642506cd6aaf45d")
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should be able to checkout the master branch from the repo", func() {
					err := repo.Checkout("master")
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})
})
