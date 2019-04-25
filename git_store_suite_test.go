package gitstore

import (
	"fmt"

	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pusher/git-store/test"
)

var repositoryPath string
var repositoryURL string
var fixturesRepoPath, _ = filepath.Abs("./fixtures/repo.tgz")

func TestGitStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "GitStore Suite", test.Reporters())
}

func setupRepository() string {
	dir, err := ioutil.TempDir("", "git-store")
	Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("tar", "-zxf", fixturesRepoPath, "-C", dir, "--strip-components", "1")
	err = cmd.Run()
	Expect(err).ToNot(HaveOccurred())

	return dir
}

func teardownRepository(dir string) {
	os.RemoveAll(dir)
}

var _ = BeforeSuite(func() {
	repositoryPath = setupRepository()
	repositoryURL = fmt.Sprintf("file://%s", repositoryPath)
})

var _ = AfterSuite(func() {
	teardownRepository(repositoryPath)
})
