package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"

	"errors"
	"net/url"
	"strings"

	"flag"

	"os"

	"time"

	"encoding/json"
	"io/ioutil"
	"sync"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

var (
	mongoSession  *mgo.Session
	mongo         *mgo.DialInfo
	workerBaseURL = "http://127.0.0.1:7777/"
)

func init() {

	uri := "mongodb://localhost:27017"
	var err error

	if d := os.Getenv("GFWORKER"); d != "" {
		workerBaseURL = d
	}

	mongo, _ = mgo.ParseURL(uri)
	mongoConnected := false
	for mongoConnected == false {
		mongoSession, err = mgo.Dial(uri)
		if err != nil {
			log.WithError(err).WithField("url", uri).Panic("Can't connect to MongoDB")
		} else {
			mongoConnected = true
		}
	}

	mongoSession.SetSafe(nil)

}
func (r *RepoConfig) Hash() string {
	h, o, n, _ := r.ParseURL()
	return MD5(h + "/" + o + "/" + n)
}
func (r *RepoConfig) ParseURL() (host, owner, name string, err error) {

	x := strings.Index(r.URL, ":")
	if x < 0 {
		err = errors.New("Bad URL")
		return
	}
	path := strings.Split(r.URL[x:], "/")
	if path[len(path)-1] == "" {
		owner = strings.ToLower(path[len(path)-3])
		name = strings.ToLower(path[len(path)-2])
	} else {
		owner = strings.ToLower(path[len(path)-2])
		name = strings.ToLower(path[len(path)-1])
	}
	if strings.Contains(r.URL, "://") {
		u, err := url.Parse(r.URL)
		if err != nil {
			return "", "", "", err
		}
		host = strings.ToLower(u.Host)
		if d := strings.Index(host, ":"); d > -1 {
			host = host[0:d]
		}

	} else {
		host = host[strings.Index(host, "@")+1 : strings.Index(host, ":")]
	}
	name = strings.TrimSuffix(name, ".git")

	return
}

/*func getemail() {
	db := mongoSession.Clone().DB("gf")
	defer db.Session.Close()
	db.C("email2login").Find(bson.M{"_id": id}).One(title)
}*/

func (r *RepoConfig) getCachedStat() *RepoStat {
	db := mongoSession.Clone().DB("gf")
	defer db.Session.Close()
	//host, owner, repo, _ := r.ParseURL()
	cachedRepo := Repo{}
	db.C("repostats").Find(bson.M{"token": r.Hash()}).One(&cachedRepo)

	if cachedRepo.Token != "" {
		return &cachedRepo.Stat
	}
	return nil
}

var tokensToFetch []string
var mu sync.Mutex

func repoQuery(repoURL string) {
	c := http.DefaultClient
	form := url.Values{"repo": {strings.Join(tokensToFetch, ",")}}
	req, _ := http.NewRequest("POST", workerBaseURL+"query", strings.NewReader(form.Encode()))
	resp, _ := c.Do(req)
	if resp.StatusCode == 200 {
		defer resp.Body.Close()
		contents, _ := ioutil.ReadAll(resp.Body)

		if len(contents) == 32 {
			mu.Lock()
			tokensToFetch = append(tokensToFetch, string(contents))
			mu.Unlock()
		}
	}
}

func tokensFetchLoop() {
	db := mongoSession.Clone().DB("gf")
	defer db.Session.Close()

	c := http.DefaultClient

	for {
		if len(tokensToFetch) > 0 {
			form := url.Values{"tokens": {strings.Join(tokensToFetch, ",")}}
			req, _ := http.NewRequest("POST", workerBaseURL+"check", strings.NewReader(form.Encode()))
			resp, err := c.Do(req)
			if resp.StatusCode == 200 {

				defer resp.Body.Close()

				decoder := json.NewDecoder(resp.Body)
				var data map[string]RepoStat

				err = decoder.Decode(&data)

				if err != nil {
					log.WithError(err).Error("can't decode repostat")
					continue
				}
				for token, rs := range data {
					db.C("repostats").Insert(bson.M{"token": token, "stat": rs})
				}
			}
		}
		time.Sleep(time.Second)
	}
}

func main() {
	worker := flag.Bool("worker", false, "Run in worker mode")

	flag.Parse()

	if *worker {
		fmt.Println("Running in worker mode")
		workerHandler()
		return
	}

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/repo", func(c *gin.Context) {
		url, _ := c.GetQuery("url")

		rc := RepoConfig{URL: url}
		rs := rc.getCachedStat()

		if rs == nil {
			c.JSON(200, gin.H{"status": "processing"})
			return
		}

		c.JSON(200, gin.H{"status": "ready", "hash": rc.Hash()})
	})

	r.GET("/rs", func(c *gin.Context) {
		url, _ := c.GetQuery("url")
		r := RepoConfig{URL: url}
		rs, err := r.Stat()
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		c.JSON(200, rs)
	})

	r.Run(":8080")
}
