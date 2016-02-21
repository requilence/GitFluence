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

	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var DOC_EXTS = []string{"md", "markdown", "mdown", "mkdn", "mdwn", "mdtxt", "txt", "text", "doc", "htm", "html"}

var DEPS_REGEXPS = []string{`(^|/)cache/`, `^[Dd]ependencies/`, `^deps/`, `^tools/`, `(^|/)configure$`, `(^|/)configure.ac$`, `(^|/)config.guess$`, `(^|/)config.sub$`, `cpplint.py`, `node_modules/`, `bower_components/`, `^rebar$`, `erlang.mk`, `Godeps/_workspace/`, `(\.|-)min\.(js|css)$`, `([^\s]*)import\.(css|less|scss|styl)$`, `(^|/)bootstrap([^.]*)\.(js|css|less|scss|styl)$`, `(^|/)custom\.bootstrap([^\s]*)(js|css|less|scss|styl)$`, `(^|/)font-awesome\.(css|less|scss|styl)$`, `(^|/)foundation\.(css|less|scss|styl)$`, `(^|/)normalize\.(css|less|scss|styl)$`, `(^|/)[Bb]ourbon/.*\.(css|less|scss|styl)$`, `(^|/)animate\.(css|less|scss|styl)$`, `third[-_]?party/`, `3rd[-_]?party/`, `vendors?/`, `extern(al)?/`, `(^|/)[Vv]+endor/`, `^debian/`, `run.n$`, `bootstrap-datepicker/`, `(^|/)jquery([^.]*)\.js$`, `(^|/)jquery\-\d\.\d+(\.\d+)?\.js$`, `(^|/)jquery\-ui(\-\d\.\d+(\.\d+)?)?(\.\w+)?\.(js|css)$`, `(^|/)jquery\.(ui|effects)\.([^.]*)\.(js|css)$`, `jquery.fn.gantt.js`, `jquery.fancybox.(js|css)`, `fuelux.js`, `(^|/)jquery\.fileupload(-\w+)?\.js$`, `(^|/)slick\.\w+.js$`, `(^|/)Leaflet\.Coordinates-\d+\.\d+\.\d+\.src\.js$`, `leaflet.draw-src.js`, `leaflet.draw.css`, `Control.FullScreen.css`, `Control.FullScreen.js`, `leaflet.spin.js`, `wicket-leaflet.js`, `.sublime-project`, `.sublime-workspace`, `(^|/)prototype(.*)\.js$`, `(^|/)effects\.js$`, `(^|/)controls\.js$`, `(^|/)dragdrop\.js$`, `(.*?)\.d\.ts$`, `(^|/)mootools([^.]*)\d+\.\d+.\d+([^.]*)\.js$`, `(^|/)dojo\.js$`, `(^|/)MochiKit\.js$`, `(^|/)yahoo-([^.]*)\.js$`, `(^|/)yui([^.]*)\.js$`, `(^|/)ckeditor\.js$`, `(^|/)tiny_mce([^.]*)\.js$`, `(^|/)tiny_mce/(langs|plugins|themes|utils)`, `(^|/)MathJax/`, `(^|/)Chart\.js$`, `(^|/)[Cc]ode[Mm]irror/(\d+\.\d+/)?(lib|mode|theme|addon|keymap|demo)`, `(^|/)shBrush([^.]*)\.js$`, `(^|/)shCore\.js$`, `(^|/)shLegacy\.js$`, `(^|/)angular([^.]*)\.js$`, `(^|\/)d3(\.v\d+)?([^.]*)\.js$`, `(^|/)react(-[^.]*)?\.js$`, `(^|/)modernizr\-\d\.\d+(\.\d+)?\.js$`, `(^|/)modernizr\.custom\.\d+\.js$`, `(^|/)knockout-(\d+\.){3}(debug\.)?js$`, `(^|/)docs?/_?(build|themes?|templates?|static)/`, `(^|/)admin_media/`, `^fabfile\.py$`, `^waf$`, `^.osx$`, `\.xctemplate/`, `\.imageset/`, `^Carthage/`, `^Pods/`, `(^|/)Sparkle/`, `Crashlytics.framework/`, `Fabric.framework/`, `gitattributes$`, `gitignore$`, `gitmodules$`, `(^|/)gradlew$`, `(^|/)gradlew\.bat$`, `(^|/)gradle/wrapper/`, `-vsdoc\.js$`, `\.intellisense\.js$`, `(^|/)jquery([^.]*)\.validate(\.unobtrusive)?\.js$`, `(^|/)jquery([^.]*)\.unobtrusive\-ajax\.js$`, `(^|/)[Mm]icrosoft([Mm]vc)?([Aa]jax|[Vv]alidation)(\.debug)?\.js$`, `^[Pp]ackages\/.+\.\d+\/`, `(^|/)extjs/.*?\.js$`, `(^|/)extjs/.*?\.xml$`, `(^|/)extjs/.*?\.txt$`, `(^|/)extjs/.*?\.html$`, `(^|/)extjs/.*?\.properties$`, `(^|/)extjs/.sencha/`, `(^|/)extjs/docs/`, `(^|/)extjs/builds/`, `(^|/)extjs/cmd/`, `(^|/)extjs/examples/`, `(^|/)extjs/locale/`, `(^|/)extjs/packages/`, `(^|/)extjs/plugins/`, `(^|/)extjs/resources/`, `(^|/)extjs/src/`, `(^|/)extjs/welcome/`, `(^|/)html5shiv\.js$`, `^[Tt]ests?/fixtures/`, `^[Ss]pecs?/fixtures/`, `(^|/)cordova([^.]*)\.js$`, `(^|/)cordova\-\d\.\d(\.\d)?\.js$`, `foundation(\..*)?\.js$`, `^Vagrantfile$`, `.[Dd][Ss]_[Ss]tore$`, `^vignettes/`, `^inst/extdata/`, `octicons.css`, `sprockets-octicons.scss`, `(^|/)activator$`, `(^|/)activator\.bat$`, `proguard.pro`, `proguard-rules.pro`, `^puphpet/`, `(^|/)\.google_apis/`}

var depRegexp *regexp.Regexp

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

func (a ByLines) Len() int      { return len(a) }
func (a ByLines) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLines) Less(i, j int) bool {
	return a[i].CodeLines.Total+a[i].DocLines.Total+a[i].TestLines.Total+a[i].Resources.Total > a[j].CodeLines.Total+a[j].DocLines.Total+a[j].TestLines.Total+a[j].Resources.Total
}

func init() {
	if !exists(REPOS_DIR) {
		os.MkdirAll(REPOS_DIR, 0777)
	}

	depRegexp = regexp.MustCompile(strings.Join(DEPS_REGEXPS, "|"))
	/*for _,re:=range DEPS_REGEXPS{
		depRegexpStr+="("
	}*/
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

var blameHeaderRE = regexp.MustCompile("([0-9a-f]{40})\\s([^(]*)?\\(<([^>]*[^0-9]*)([^\\s]*\\s[^\\s]*\\s[^\\s]*)(\\s*[0-9]*\\))\\s[^\\n]*")

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

	if depRegexp.MatchString(filePath) {
		return nil, errors.New("File is dependence")
	}

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
		//TODO: binary file?

		return nil, errors.New("Bad git-blame output")
	}
	ext := path.Ext(filePath)

	if len(ext) < 2 {
		fs.IsDoc = true
		// may be a binary file. will check later
	} else {
		ext = strings.ToLower(ext[1:])
		for _, docExt := range DOC_EXTS {
			if ext == docExt {
				fs.IsDoc = true
				break
			}
		}
	}

	if !fs.IsDoc {
		//TODO: handle specific test file cases
		if strings.Contains(filePath, "test") {
			fs.IsTest = true
		}
	}

	blameHeader := blameHeaderRE.FindStringSubmatch(string(out[0:200]))

	if len(blameHeader) < 6 {
		return nil, errors.New("bad row: " + string(out[0:200]))
	}
	l1 := len(blameHeader[1])
	l2 := len(blameHeader[2])
	l3 := len(blameHeader[3])
	l4 := len(blameHeader[4])
	l5 := len(blameHeader[5])

	spew.Dump(blameHeader)

	pos := 0
	now := time.Now().Unix()
	c := 0
	commentStarted := false

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

		c = c + l1 + l2 + l3 + l4 + l5

		lineStart := c
		for out[c] != '\n' {
			c++
		}
		lineIsComment := false
		if !fs.IsDoc {
			line := string(out[lineStart:c])

			//TODO: other language comments handling
			if !commentStarted {
				first2Symbol := strings.TrimSpace(line)[0:1]

				if first2Symbol == "//" {
					lineIsComment = true
				} else if first2Symbol == "/*" {
					commentStarted = true
					lineIsComment = true
				} else if first2Symbol[0:0] == "#" {
					lineIsComment = true
				}
			} else {
				lineIsComment = true
				if strings.Contains(line, "*/") {
					commentStarted = false
				}
			}

		}
		var linesStat *LinesStat
		if lineIsComment || fs.IsDoc {
			linesStat = &fs.Users[email].DocLines
		} else if fs.IsTest {
			linesStat = &fs.Users[email].TestLines
		} else {
			linesStat = &fs.Users[email].CodeLines
		}

		linesStat.Total++
		if daysAfterCommit < 366 {
			linesStat.LastYear++
			if daysAfterCommit < 184 {
				linesStat.Last6Month++
				if daysAfterCommit < 93 {
					linesStat.Last3Month++
					if daysAfterCommit < 32 {
						linesStat.LastMonth++
					}
				}
			}
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
				log.WithError(err).WithField("file", file).Error("BlameFile returned error")
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
			rs.CodeLines.Append(us.CodeLines)
			rs.TestLines.Append(us.TestLines)
			rs.DocLines.Append(us.DocLines)
			rs.Resources.Append(us.Resources)

			rs.Users[email].CodeLines.Append(us.CodeLines)
			rs.Users[email].TestLines.Append(us.TestLines)
			rs.Users[email].DocLines.Append(us.DocLines)
			rs.Users[email].Resources.Append(us.Resources)

			if _, exists := rs.Users[email].LinesPerExt[ext]; !exists {
				rs.Users[email].LinesPerExt[ext] = &LinesStat{}
			}

			rs.Users[email].LinesPerExt[ext].Append(us.CodeLines)
			rs.Users[email].LinesPerExt[ext].Append(us.TestLines)
			rs.Users[email].LinesPerExt[ext].Append(us.DocLines)
			rs.Users[email].LinesPerExt[ext].Append(us.Resources)

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
		fmt.Printf("%d, %v: \n", i, user.Username)
		spew.Dump(user)
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
		body, _ := ioutil.ReadAll(c.Request.Body)

		tokens := strings.Split(string(body), ",")
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
		body, _ := ioutil.ReadAll(c.Request.Body)

		repoURL := string(body)
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
