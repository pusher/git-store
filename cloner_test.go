package gitstore

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

var _ = Describe("GitStore", func() {
	var rs *RepoStore

	BeforeEach(func() {
		rs = NewRepoStore()
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
})
