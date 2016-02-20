package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"crypto/md5"
	"encoding/hex"

	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const REPOS_DIR = "/tmp/repos"
const TOP_REPO_USERS = 15

func MD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
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

type ByLines []*UserStat

func (a ByLines) Len() int           { return len(a) }
func (a ByLines) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLines) Less(i, j int) bool { return a[i].Lines.Total > a[j].Lines.Total }

func init() {
	if !exists(REPOS_DIR) {
		os.MkdirAll(REPOS_DIR, 0777)
	}

}

func (repo RepoConfig) GitClone() (string, error) {
	host, owner, name, err := repo.ParseURL()

	if err != nil {
		return "", err
	}

	repoDst := REPOS_DIR + "/" + host + "/" + owner + "/" + name

	if exists(repoDst) {
		return repoDst, nil
	} else {
		cmd := exec.Command("git", "clone", repo.URL, repoDst)
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

func GithubUsernameFromAPI(owner, repo, commitID string) (string, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "cfb633bfe4d5bdbd202c023cfa092fc4e9d3a45c"},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	commit, _, err := client.Repositories.GetCommit(owner, repo, commitID)
	//rl, _, _ := client.RateLimits()
	//	fmt.Printf("RAte limits %v, %v", rl.Core, rl.Search)
	if err != nil {
		return "", err
	}
	return *commit.Author.Login, nil
}

func GithubUsername(owner, repo, commitID string) string {
	login, err := GithubUsernameFromAPI(owner, repo, commitID)

	if err != nil {
		fmt.Printf("GithubUsernameByCommitFromAPI: %v", err.Error())
	}
	return login

}
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
		if fs.Users[email].CommitDays < daysAfterCommit {
			fs.Users[email].CommitID = string(out[c : c+l1])
			fs.Users[email].CommitDays = daysAfterCommit
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

func (r *RepoConfig) Stat() (*RepoStat, error) {
	repoPath, err := r.GitClone()
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
	fmt.Printf("CPUs: %d\n", runtime.NumCPU())
	pool := make(chan bool, runtime.NumCPU())
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

		wg.Add(1)
		wgC++

		pool <- true
		go func(mu *sync.Mutex, repoPath string, file string, rs *RepoStat) {
			fmt.Printf("File: %v\n", file)
			defer func() {
				<-pool
				wg.Done()
			}()
			fs, err := BlameFile(repoPath, file)

			if err != nil {
				fmt.Printf("error: %v", err.Error())
			} else {
				if fs != nil && fs.TotalLines > 0 {
					func() {
						mu.Lock()
						defer mu.Unlock()
						rs.Files[file] = fs

					}()

				}

			}
		}(&mu, repoPath, file, &rs)

	}
	wg.Wait()
	usersSlice := []*UserStat{}
	rs.Users = make(map[string]*UserStat)
	for file, fs := range rs.Files {
		ext := path.Ext(file)

		for email, us := range fs.Users {
			if _, exists := rs.Users[email]; !exists {
				us := UserStat{}
				rs.Users[email] = &us
				usersSlice = append(usersSlice, &us)
				rs.Users[email].Email = email
				rs.Users[email].LinesPerExt = make(map[string]*LinesStat)
			}
			if rs.Users[email].CommitDays < us.CommitDays {
				rs.Users[email].CommitID = us.CommitID
				rs.Users[email].CommitDays = us.CommitDays
			}
			rs.Lines.Append(us.Lines)
			rs.Users[email].Lines.Append(us.Lines)

			if _, exists := rs.Users[email].LinesPerExt[ext]; !exists {
				rs.Users[email].LinesPerExt[ext] = &LinesStat{}
			}

			rs.Users[email].LinesPerExt[ext].Append(us.Lines)
		}
	}

	sort.Sort(ByLines(usersSlice))

	f := strings.Split(repoPath, "/")
	owner := f[len(f)-2]
	repo := f[len(f)-1]

	maxUsers := TOP_REPO_USERS
	if len(usersSlice) < TOP_REPO_USERS {
		maxUsers = len(usersSlice)
	}
	for i, user := range usersSlice[0:maxUsers] {
		user.Username = GithubUsername(owner, repo, user.CommitID)
		fmt.Printf("%d, %v: %d\n", i, user.Username, user.Lines.Total)
	}
	return &rs, nil

}

var tasks chan string
var readyRepos = make(map[string]*RepoStat)

func workerLoop() {
	tasks = make(chan string, 65536)
	for {
		repoURL := <-tasks
		fmt.Printf("Received task: %v\n", repoURL)
		repo := RepoConfig{URL: repoURL}

		hash := repo.Hash()

		if _, exists := readyRepos[hash]; exists {
			continue
		}

		rs, err := repo.Stat()

		if err != nil {
			log.WithError(err).Error("Can't fetch repostat")
			continue
		}

		readyRepos[hash] = rs
	}
}

func workerHandler() {

	go workerLoop()
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.POST("/check", func(c *gin.Context) {
		r, _ := c.GetPostForm("tokens")
		tokens := strings.Split(r, ",")
		if len(tokens) > 0 {
			res := gin.H{}

			for _, token := range tokens {
				if v, exists := readyRepos[token]; exists {
					res[token] = v
				}
			}
			if len(res) > 0 {
				c.JSON(200, res)
			} else {
				c.AbortWithStatus(http.StatusNoContent)
			}
		} else {
			c.AbortWithStatus(http.StatusNoContent)
		}
		return
	})

	r.POST("/query", func(c *gin.Context) {
		repoURL := c.PostForm("repo")
		fmt.Println("repo: " + repoURL)
		r := RepoConfig{URL: repoURL}

		hash := r.Hash()
		if _, exists := readyRepos[hash]; !exists {
			tasks <- repoURL
		}
		c.String(200, hash)
		return
	})

	r.Run(":7777")
	/*
		repoStat, err := jobs.RegisterType("repoStat", 3, func(repoURL string) error {
			msg := fmt.Sprintf("Hello, %s! Thanks for signing up for foo.com.", user.Name)
			if err := emails.Send(user.EmailAddress, msg); err != nil {
				// The returned error will be captured by a worker, which will then log the error
				// in the database and trigger up to 3 retries.
				return err
			}
		})
	*/
}
