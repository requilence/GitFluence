package worker

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const REPOS_DIR = "/tmp/repos"

type UserStat struct {
	Lines    LinesStat
	Email    string
	Username string
}

type LinesStat struct {
	LastMonth  int
	Last3Month int
	Last6Month int
	LastYear   int
	Total      int
}

type FileStat struct {
	TotalLines int
	Users      map[string]*UserStat
}

type RepoStat struct {
	Lines LinesStat
	Users map[string]*UserStat
	Files map[string]*FileStat
}

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

func isDir(path string) bool {
	s, err := os.Stat(path)
	return err == nil && s.IsDir()
}

func RepoListFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-z", "-r", "HEAD", "--name-only")
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	lines, err := cmd.Output()

	if err != nil {
		return nil, err
	}
	paths := strings.Split(string(lines), "\x00")

	var files []string
	for _, f := range paths {
		if !isDir(filepath.Join(repoPath, f)) {
			files = append(files, f)
		}
	}

	return files, nil
}

var blameHeaderRE = regexp.MustCompile("([0-9a-f]{40})\\s([^\\(]*)?\\(<([^>]*[^0-9]*)([^\\s]*\\s[^\\s]*\\s[^\\s]*)\\s*([0-9]*)\\)\\s([^\n]*)")

func BlameFile(repoPath string, filePath string) (*FileStat, error) {
	cmd := exec.Command("git", "blame", "-w", "-M", "-l", "-C", "-e", "--", filePath)
	fs := FileStat{Users: make(map[string]*UserStat)}
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	if len(out) < 1 {

		fileInfo, err := os.Stat(filepath.Join(repoPath, filePath))
		if err != nil {
			return nil, err
		}
		if fileInfo.Size() == 0 {
			return nil, nil
		}

		return nil, errors.New("Bad git-blame output")
	}

	blameHeader := blameHeaderRE.FindStringSubmatch(string(out[0:200]))

	l1 := len(blameHeader[1])
	l2 := len(blameHeader[2])
	l3 := len(blameHeader[3])

	pos := 0
	now := time.Now().Unix()
	c := 0

	for c < len(out) {

		/// start the new line
		fs.TotalLines++
		pos = l1 + l2

		for out[c+pos] != '>' {
			pos++
		}
		email := string(out[c+l1+l2+3 : c+pos])
		//fmt.Println("email: "+email)
		pos = l1 + l2 + l3

		date, _ := time.Parse("2006-01-02 15:04:05 -0700", string(out[c+pos+3:c+pos+3+25]))
		daysAfterCommit := int(float64((now - date.Unix())) / (60 * 60 * 24))
		//fmt.Printf("%v %v, %d\n", string(out[c+pos+3:c+pos+3+25]), date.Unix(), daysAfterCommit)

		if _, exists := fs.Users[email]; !exists {
			fs.Users[email] = &UserStat{}
		}

		fs.Users[email].Lines.Total++
		if daysAfterCommit < 366 {
			fs.Users[email].Lines.LastYear++
			if daysAfterCommit < 184 {
				fs.Users[email].Lines.Last6Month++
				if daysAfterCommit < 93 {
					fs.Users[email].Lines.Last3Month++
					if daysAfterCommit < 32 {
						fs.Users[email].Lines.LastMonth++
					}
				}
			}
		}
		for out[c] != '\n' {
			c++
		}
		c++

	}
	return &fs, nil
}

func GetRepoStat(url string) (*RepoStat, error) {
	repoPath, err := GitClone(url)
	if err != nil {
		return nil, err
	}
	var repoStat *RepoStat
	repoStat, err = BlameRepo(repoPath)
	if err != nil {
		return nil, err
	}

	return repoStat, nil
}

func BlameRepo(repoPath string) (*RepoStat, error) {

	rs := RepoStat{}
	rs.Files = make(map[string]*FileStat)
	files, err := RepoListFiles(repoPath)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex

	wg := sync.WaitGroup{}
	wgC := 0

	for _, file := range files {

		file := string(file)
		if file == "" {
			continue
		}
		//TODO: more vendor dirs

		if strings.HasPrefix(file, "vendor/") {
			continue
		}
		if strings.HasPrefix(file, "pkg") {
			continue
		}
		if strings.HasPrefix(file, "bin") {
			continue
		}
		fmt.Printf("File: %v\n", file)
		wg.Add(1)
		wgC++

		go func(mu *sync.Mutex, repoPath string, file string, rs *RepoStat) {
			defer wg.Done()
			fs, err := BlameFile(repoPath, file)

			if err != nil {
				fmt.Printf("error: %v", err.Error())
			} else {
				if fs != nil && fs.TotalLines > 0 {
					rs.Lines.Total += fs.TotalLines
					func() {
						mu.Lock()
						defer mu.Unlock()
						rs.Files[file] = fs

					}()

				}

			}
		}(&mu, repoPath, file, &rs)
		if wgC > 32 {
			wgC = 0
			wg.Wait()
		}

	}
	wg.Wait()

	return &rs, nil

}
