package gitstore

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var expectedFoo = `package main

import "fmt"

func main() {
	fmt.Println("Hello, playground")
}
`

var _ = Describe("GitStore", func() {

	Context("When the repository is cloned asynchronously", func() {
		var rs *RepoStore
		var rc *AsyncRepoCloner
		var done <-chan struct{}

		BeforeEach(func() {
			rs = NewRepoStore()
			var err error
			rc, done, err = rs.GetAsync(&RepoRef{
				URL: repositoryURL,
			})
			Expect(err).ToNot(HaveOccurred())
			Eventually(done, 5*time.Second).Should(BeClosed())
			Expect(rc.Ready).To(BeTrue())
			err = rc.Repo.Checkout("master")
			Expect(err).ToNot(HaveOccurred())
			Expect(rc).ToNot(BeNil())
		})

		It("Should checkout the commit", func() {
			err := rc.Repo.Checkout("b029517f6300c2da0f4b651b8642506cd6aaf45d")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When the git repository is cloned", func() {
		var rs *RepoStore
		var repo *Repo

		BeforeEach(func() {
			rs = NewRepoStore()
			var err error
			repo, err = rs.Get(&RepoRef{
				URL: repositoryURL,
			})
			Expect(err).ToNot(HaveOccurred())
			err = repo.Checkout("master")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should be able to count all files without error.", func() {
			files, err := repo.GetAllFiles("", true)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(9))
		})

		Context("with symlinks enabled", func() {
			It("Should be able to count all files without error.", func() {
				files, err := repo.GetAllFiles("", false)
				Expect(err).ToNot(HaveOccurred())
				Expect(files).To(HaveLen(11))
			})
		})

		It("Should be able to checkout the second commit ref without error", func() {
			err := repo.Checkout("b8e471f58bcbca63b07bda20e428190409c2db47")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should be able to fetch CHANGELOG file from map", func() {
			files, err := repo.GetAllFiles("", true)
			Expect(err).ToNot(HaveOccurred())
			changelog, ok := files["CHANGELOG"]
			Expect(ok).To(BeTrue())
			Expect(changelog.Contents()).To(Equal("Initial changelog\n"))
		})

		Context("and the first commit is checked out", func() {
			BeforeEach(func() {
				err := repo.Checkout("b029517f6300c2da0f4b651b8642506cd6aaf45d")
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should be able to read the license file", func() {
				license, err := repo.GetFile("LICENSE")
				Expect(err).ToNot(HaveOccurred())
				Expect(license).ToNot(BeNil())
			})

			It("Should not be able to read CHANGELOG file (does not exist)", func() {
				changelog, err := repo.GetFile("CHANGELOG")
				Expect(err).To(HaveOccurred())
				Expect(changelog).To(BeNil())
			})

			It("Should be able to get the LastUpdated timestamp", func() {
				lastUpdated, err := repo.LastUpdated()
				Expect(err).ToNot(HaveOccurred())
				utcPlus2 := time.FixedZone("+0200", int(2*time.Hour.Seconds()))
				expectedTime := time.Date(2015, time.March, 31, 13, 42, 21, 0, utcPlus2)
				Expect(lastUpdated).To(BeTemporally("==", expectedTime))
			})
		})

		Context("and the eighth commit is checkout out", func() {
			BeforeEach(func() {
				err := repo.Checkout("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should be able to read CHANGELOG file.", func() {
				changelog, err := repo.GetFile("CHANGELOG")
				Expect(err).ToNot(HaveOccurred())
				Expect(changelog).ToNot(BeNil())
			})

			It("Should be able to read the vendor/foo.go file", func() {
				files, err := repo.GetAllFiles("", true)
				Expect(err).ToNot(HaveOccurred())
				foo, ok := files["vendor/foo.go"]
				Expect(ok).To(BeTrue())
				Expect(foo.Contents()).To(Equal(expectedFoo))
			})

			It("Should be able to get the LastUpdated timestamp", func() {
				lastUpdated, err := repo.LastUpdated()
				Expect(err).ToNot(HaveOccurred())
				utcPlus2 := time.FixedZone("+0200", int(2*time.Hour.Seconds()))
				expectedTime := time.Date(2015, time.April, 5, 23, 30, 47, 0, utcPlus2)
				Expect(lastUpdated).To(BeTemporally("==", expectedTime))
			})

			Context("the IsFile method", func() {
				It("Should correctly identify regular files", func() {
					isFile, err := repo.IsFile("CHANGELOG")
					Expect(err).ToNot(HaveOccurred())
					Expect(isFile).To(BeTrue())
				})

				It("Should correctly identify non-regular files", func() {
					isFile, err := repo.IsFile("vendor")
					Expect(err).ToNot(HaveOccurred())
					Expect(isFile).To(BeFalse())
				})

				It("Should throw an error if an invalid path is given", func() {
					_, err := repo.IsFile("not-valid")
					Expect(err).To(HaveOccurred())
				})
			})

			Context("the IsDirectory method", func() {
				It("Should correctly identify directories", func() {
					isFile, err := repo.IsDirectory("vendor")
					Expect(err).ToNot(HaveOccurred())
					Expect(isFile).To(BeTrue())
				})

				It("Should correctly identify non-directories", func() {
					isFile, err := repo.IsDirectory("CHANGELOG")
					Expect(err).ToNot(HaveOccurred())
					Expect(isFile).To(BeFalse())
				})

				It("Should throw an error if an invalid path is given", func() {
					_, err := repo.IsDirectory("not-valid")
					Expect(err).To(HaveOccurred())
				})
			})

			var findsFiles = func(path string, count int) {
				It(fmt.Sprintf("Finds %d files inside path %s", count, path), func() {
					files, err := repo.GetAllFiles(path, true)
					Expect(err).ToNot(HaveOccurred())
					Expect(files).To(HaveLen(count))
				})
			}

			findsFiles("", 9)
			findsFiles("**/*.go", 2)
			findsFiles("**/*.json", 2)
			findsFiles("json/*", 2)
			findsFiles("vendor/*", 1)
		})

	})
})
