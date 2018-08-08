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

package store

import (
	"fmt"
	"regexp"
	"strings"
)

type urlType int

const (
	_               = iota
	httpURL urlType = iota + 1
	sshURL
	fileURL
	gitURL
	rsyncURL
)

const gitRegex = "((git|ssh|file|rsync|http(s)?)|((\\w+[\\:\\w]+?@)?[\\w\\.]+))(:(//)?)(\\w+[\\:\\w]+?@)?([\\w\\.\\:/\\-~]+)(\\.git)?(/)?"

// RepoRef contains information required to construct a Git Repository
type RepoRef struct {
	URL             string
	user            string
	pass            string
	urlType         urlType
	SecretName      string
	SecretNamespace string
}

// Validate will parse input URL and update private fields of the reference
func (r *RepoRef) Validate() error {
	// Does the URL pass basic validation
	valid, err := validGitURL(r.URL)
	if err != nil {
		return fmt.Errorf("unable to validate URL: %v", err)
	}
	if !valid {
		return fmt.Errorf("invalid git url: %s", r.URL)
	}

	// Extract repository type, user and password from URL
	repoType, user, pass, err := getRepoTypeAndUser(r.URL)
	if err != nil {
		return fmt.Errorf("unable to determine repository type: %v", err)
	}
	r.urlType = repoType
	r.user = user
	r.pass = pass
	return nil
}

// validGitURL checks that the input URL passes the basic URL regex
func validGitURL(url string) (bool, error) {
	r, err := regexp.Compile(gitRegex)
	if err != nil {
		return false, fmt.Errorf("unable to compile regex: %v", err)
	}
	return r.MatchString(url), nil
}

// getRepoTypeAndUser determines what kind of repository is being clones and
// extracts user/pass information from the string
func getRepoTypeAndUser(url string) (urlType, string, string, error) {
	r, err := regexp.Compile(gitRegex)
	if err != nil {
		return 0, "", "", fmt.Errorf("unable to compile regex: %v", err)
	}

	// Fetch regex groups
	matches := r.FindStringSubmatch(url)
	if len(matches) != 12 {
		return 0, "", "", fmt.Errorf("should have matched 12 capture groups, matched %d", len(matches))
	}

	// Parse username from regex capture groups
	user, pass := parseUserPassFromMatches(matches)
	git := "git"

	// Switch on protocol prefixes
	switch matches[1] {
	case "ssh":
		if user == "" {
			user = git
		}
		return sshURL, user, pass, nil
	case "http":
		return httpURL, user, pass, nil
	case "https":
		return httpURL, user, pass, nil
	case "file":
		return fileURL, user, pass, nil
	case "rsync":
		return rsyncURL, user, pass, nil
	case git:
		if user == "" {
			user = git
		}
		return gitURL, user, pass, nil
	}

	// SSH only beyond this point
	if user == "" {
		user = git
	}

	// SSH URL with username x@y.com:foo/bar
	if strings.Contains(matches[0], "@") {
		return sshURL, user, pass, nil
	}

	// SSH URL without username y.com:foo/bar
	if matches[6] == ":" {

		return sshURL, user, pass, nil
	}
	return 0, user, pass, fmt.Errorf("unable to determine repository type")
}

// parseUserPassFromMatches splits the user:pass@ strings from the regex group
func parseUserPassFromMatches(matches []string) (string, string) {
	var userPass string
	if matches[5] != "" {
		userPass = strings.TrimRight(matches[5], "@")
	}
	if matches[8] != "" {
		userPass = strings.TrimRight(matches[8], "@")
	}

	if strings.Contains(userPass, ":") {
		split := strings.Split(userPass, ":")
		return split[0], split[1]
	}
	return userPass, ""
}
