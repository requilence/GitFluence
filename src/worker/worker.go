package worker

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

const REPOS_DIR = "/tmp/repos"

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func init() {
	if !exists(REPOS_DIR) {
		os.MkdirAll(REPOS_DIR, 0777)
	}

}

func GitClone(repoURL string) (string, error) {
	x := strings.Index(repoURL, ":")
	if x < 0 {
		return "", errors.New("Bad URL")
	}
	path := strings.Split(repoURL[x:], "/")
	var user, repo, host string
	if path[len(path)-1] == "" {
		user = strings.ToLower(path[len(path)-3])
		repo = strings.ToLower(path[len(path)-2])
	} else {
		user = strings.ToLower(path[len(path)-2])
		repo = strings.ToLower(path[len(path)-1])
	}
	if strings.Contains(repoURL, "://") {
		u, err := url.Parse(repoURL)
		if err != nil {
			return "", err
		}
		host = u.Host
		if d := strings.Index(host, ":"); d > -1 {
			host = host[0:d]
		}

	} else {
		host = host[strings.Index(host, "@")+1 : strings.Index(host, ":")]
	}

	strings.TrimSuffix(repo, ".git")

	repoDst := REPOS_DIR + "/" + host + "/" + user + "/" + repo

	if exists(repoDst) {
		return repoDst, nil
	} else {
		cmd := exec.Command("git", "clone", repoURL, repoDst)
		cmd.Stderr = os.Stderr
		_, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return repoDst, nil
	}

}
