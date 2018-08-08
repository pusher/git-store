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

import "testing"

func TestGetRepoType(t *testing.T) {
	typetests := []struct {
		in       string
		user     string
		pass     string
		repoType urlType
	}{
		{"ssh://user@host.xz:port/path/to/repo.git/", "user", "", sshURL},
		{"ssh://user@host.xz/path/to/repo.git/", "user", "", sshURL},
		{"ssh://host.xz:port/path/to/repo.git/", "git", "", sshURL},
		{"ssh://host.xz/path/to/repo.git/", "git", "", sshURL},
		{"ssh://user@host.xz/path/to/repo.git/", "user", "", sshURL},
		{"ssh://user:pass@host.xz/path/to/repo.git/", "user", "pass", sshURL},
		{"ssh://host.xz/path/to/repo.git/", "git", "", sshURL},
		{"ssh://user@host.xz/~user/path/to/repo.git/", "user", "", sshURL},
		{"ssh://host.xz/~user/path/to/repo.git/", "git", "", sshURL},
		{"ssh://user@host.xz/~/path/to/repo.git", "user", "", sshURL},
		{"ssh://host.xz/~/path/to/repo.git", "git", "", sshURL},
		{"user@host.xz:/path/to/repo.git/", "user", "", sshURL},
		{"host.xz:/path/to/repo.git/", "git", "", sshURL},
		{"user@host.xz:~user/path/to/repo.git/", "user", "", sshURL},
		{"host.xz:~user/path/to/repo.git/", "git", "", sshURL},
		{"user@host.xz:path/to/repo.git", "user", "", sshURL},
		{"user:pass@host.xz:path/to/repo.git", "user", "pass", sshURL},
		{"host.xz:path/to/repo.git", "git", "", sshURL},
		{"rsync://host.xz/path/to/repo.git/", "", "", rsyncURL},
		{"git://host.xz/path/to/repo.git/", "git", "", gitURL},
		{"git://host.xz/~user/path/to/repo.git/", "git", "", gitURL},
		{"http://host.xz/path/to/repo.git/", "", "", httpURL},
		{"https://host.xz/path/to/repo.git/", "", "", httpURL},
		{"http://user@host.xz/path/to/repo.git/", "user", "", httpURL},
		{"http://user:pass@host.xz/path/to/repo.git/", "user", "pass", httpURL},
		{"file:///path/to/repo.git/", "", "", fileURL},
		{"file://~/path/to/repo.git/", "", "", fileURL},
	}

	// Test each of the test cases above for correct output
	for _, test := range typetests {
		t.Run(test.in, func(t *testing.T) {
			repoType, user, pass, err := getRepoTypeAndUser(test.in)
			if err != nil {
				t.Errorf("error in regex: %v", err)
			}
			if repoType != test.repoType {
				t.Errorf("got %q, want %q", repoType, test.repoType)
			}
			if user != test.user {
				t.Errorf("got %q, want %q", user, test.user)
			}
			if pass != test.pass {
				t.Errorf("got %q, want %q", pass, test.pass)
			}
		})
	}
}
