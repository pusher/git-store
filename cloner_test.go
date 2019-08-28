package gitstore

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

var _ = Describe("GitStore", func() {
	var cloneTests = func(path string) {
		var rs *RepoStore
		BeforeEach(func() {
			rs = NewRepoStore(path)
		})

		Context("when a new repo is cloned", func() {
			var repo *Repo

			BeforeEach(func() {
				var err error
				repo, err = rs.Get(&RepoRef{
					URL: repositoryURL,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should checkout a detached HEAD", func() {
				ref, err := repo.repository.Head()
				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Name().String()).ToNot(Equal("refs/heads/master"))
			})

			It("should have no local branches checked out", func() {
				cfg, err := repo.repository.Storer.Config()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg.Branches).To(BeEmpty())
			})

			It("refs/heads/master should not be a resolvable reference", func() {
				ref, err := storer.ResolveReference(repo.repository.Storer, plumbing.ReferenceName("refs/heads/master"))
				Expect(err).To(HaveOccurred())
				Expect(ref).To(BeNil())
			})
		})

		Context("when a repo has a non-master branch", func() {
			var repositoryDir string

			BeforeEach(func() {
				var err error
				repositoryDir, err = ioutil.TempDir("", "git-store")
				Expect(err).ToNot(HaveOccurred())

				cmd := exec.Command("tar", "-zxf", fixturesRepoPath, "-C", repositoryDir, "--strip-components", "1")
				err = cmd.Run()
				Expect(err).ToNot(HaveOccurred())

				for _, branch := range []string{"staging", "production", "foo", "bar", "head"} {
					cmd := exec.Command("git", "-C", repositoryDir, "checkout", "-b", branch)
					err = cmd.Run()
					Expect(err).ToNot(HaveOccurred())
				}
			})

			AfterEach(func() {
				os.RemoveAll(repositoryDir)
			})

			It("can be cloned without errors", func() {
				url := fmt.Sprintf("file://%s", repositoryDir)
				_, err := rs.Get(&RepoRef{
					URL: url,
				})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	}

	Context("(In Memory)", func() {
		cloneTests("")
	})

	Context("(On Disk)", func() {
		var tmpDir string

		BeforeEach(func() {
			// Create a temp directory for each test
			var err error
			tmpDir, err = ioutil.TempDir("", "git-store")
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		cloneTests(tmpDir)

		Context("when cloning a repo to a filesystem", func() {
			BeforeEach(func() {
				rs := NewRepoStore(tmpDir)
				_, err := rs.Get(&RepoRef{
					URL: repositoryURL,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should update the HEAD file in the .git folder to the detached ref", func() {
				head, err := ioutil.ReadFile(filepath.Join(tmpDir, repositoryURL, ".git", "HEAD"))
				Expect(err).ToNot(HaveOccurred())
				Expect(string(head)).ToNot(Equal("ref: refs/heads/master"))
			})

			It("should not have any local refs", func() {
				files, err := ioutil.ReadDir(filepath.Join(tmpDir, repositoryURL, ".git", "refs", "heads"))
				Expect(err).ToNot(HaveOccurred())
				Expect(files).To(HaveLen(0))
			})

			It("should be able to do it multiple times", func() {
				for i := 0; i <= 3; i++ {
					rs := NewRepoStore(tmpDir)
					_, err := rs.Get(&RepoRef{
						URL: repositoryURL,
					})
					Expect(err).ToNot(HaveOccurred())
				}
			})
		})
	})
})
