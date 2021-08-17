package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
	"strings"

	//"github.com/google/go-github/v38/github"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
)

type GhHosts map[string]GhHostsEntry

type GhHostsEntry struct {
	User        string
	OAuthToken  string `yaml:"oauth_token"`
	GitProtocol string `yaml:"git_protocol"`
}

func getToken() string {
	homeDir, _ := os.UserHomeDir()
	ghHostsFile := path.Join(homeDir, ".config", "gh", "hosts.yml")
	ghHostsText, _ := ioutil.ReadFile(ghHostsFile)

	var ghHosts = GhHosts{}

	_ = yaml.Unmarshal(ghHostsText, &ghHosts)
	return ghHosts["github.com"].OAuthToken
}

type Client struct {
	token  string
	ctx    context.Context
	client *github.Client
}

func (c *Client) LoadToken(token string) {
	if len(strings.Trim(token, " ")) == 0 {
		c.token = getToken()
	} else {
		c.token = token
	}
}

func (c *Client) InitClient() {
	c.ctx = context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.token})
	tc := oauth2.NewClient(c.ctx, ts)

	c.client = github.NewClient(tc)
}

func (c *Client) ValidateToken() error {
	_, _, err := c.client.Users.Get(c.ctx, "")
	return err
}

func (c *Client) FetchCommitStatus(owner string, repo string, rev string) string {
	commit, _, _ := c.client.Repositories.GetCommit(c.ctx, owner, repo, rev, nil)
	fmt.Println(*commit.SHA)
	return *commit.SHA
}
