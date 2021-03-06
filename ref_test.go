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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("GitStore", func() {

	Context("When parsing a reference", func() {
		var validateOutput = func(in string, expectedUser string, expectedPass string, expectedRepoType urlType) {
			It("Should set correct user, pass and repoType", func() {
				repoType, user, pass, err := getRepoTypeAndUser(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(repoType).To(Equal(expectedRepoType))
				Expect(user).To(Equal(expectedUser))
				Expect(pass).To(Equal(expectedPass))
			})
		}

		validateOutput("ssh://user@host.xz:port/path/to/repo.git/", "user", "", sshURL)
		validateOutput("ssh://user@host.xz/path/to/repo.git/", "user", "", sshURL)
		validateOutput("ssh://host.xz:port/path/to/repo.git/", "git", "", sshURL)
		validateOutput("ssh://host.xz/path/to/repo.git/", "git", "", sshURL)
		validateOutput("ssh://user@host.xz/path/to/repo.git/", "user", "", sshURL)
		validateOutput("ssh://user:pass@host.xz/path/to/repo.git/", "user", "pass", sshURL)
		validateOutput("ssh://host.xz/path/to/repo.git/", "git", "", sshURL)
		validateOutput("ssh://user@host.xz/~user/path/to/repo.git/", "user", "", sshURL)
		validateOutput("ssh://host.xz/~user/path/to/repo.git/", "git", "", sshURL)
		validateOutput("ssh://user@host.xz/~/path/to/repo.git", "user", "", sshURL)
		validateOutput("ssh://host.xz/~/path/to/repo.git", "git", "", sshURL)
		validateOutput("user@host.xz:/path/to/repo.git/", "user", "", sshURL)
		validateOutput("host.xz:/path/to/repo.git/", "git", "", sshURL)
		validateOutput("user@host.xz:~user/path/to/repo.git/", "user", "", sshURL)
		validateOutput("host.xz:~user/path/to/repo.git/", "git", "", sshURL)
		validateOutput("user@host.xz:path/to/repo.git", "user", "", sshURL)
		validateOutput("user:pass@host.xz:path/to/repo.git", "user", "pass", sshURL)
		validateOutput("host.xz:path/to/repo.git", "git", "", sshURL)
		validateOutput("rsync://host.xz/path/to/repo.git/", "", "", rsyncURL)
		validateOutput("git://host.xz/path/to/repo.git/", "git", "", gitURL)
		validateOutput("git://host.xz/~user/path/to/repo.git/", "git", "", gitURL)
		validateOutput("http://host.xz/path/to/repo.git/", "", "", httpURL)
		validateOutput("https://host.xz/path/to/repo.git/", "", "", httpURL)
		validateOutput("http://user@host.xz/path/to/repo.git/", "user", "", httpURL)
		validateOutput("http://user:pass@host.xz/path/to/repo.git/", "user", "pass", httpURL)
		validateOutput("file:///path/to/repo.git/", "", "", fileURL)
		validateOutput("file://~/path/to/repo.git/", "", "", fileURL)
	})

	Context("When validating a repository", func() {
		Context("with basic auth", func() {
			var validateOutput = func(url, user, pass string, matcher types.GomegaMatcher) {
				r := &RepoRef{
					URL:  url,
					User: user,
					Pass: pass,
				}
				Expect(r.Validate()).To(matcher)
			}

			It("Should allow both username and password set", func() {
				validateOutput("http://example.com", "foo", "bar", BeNil())
			})

			It("Should disallow user but no password set", func() {
				validateOutput("http://example.com", "foo", "", Not(BeNil()))
			})

			It("Should disallow password but no user set", func() {
				validateOutput("http://example.com", "", "bar", Not(BeNil()))
			})

			It("Should allow no credentials set", func() {
				validateOutput("http://example.com", "", "", BeNil())
			})
		})

		Context("with ssh auth", func() {
			It("Should disallow an empty private key", func() {
				r := &RepoRef{
					URL: "ssh://git@example.com",
				}
				Expect(r.Validate()).NotTo(BeNil())
			})
		})
	})
})
