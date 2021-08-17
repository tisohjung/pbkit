package cmd

import (
	"fmt"
	"github.com/riiid/pbkit/cli/pollapo-go/misc/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"regexp"
)

func getConfigDir() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".config", "pollapo")
}

func getCacheDir() string {
	configDir := getConfigDir()
	return path.Join(configDir, "cache")
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := github.Client{}
		var token, _ = cmd.Flags().GetString("token")
		client.LoadToken(token)
		client.InitClient()
		err := client.ValidateToken()
		if err != nil {
			return err
		}

		var config, _ = cmd.Flags().GetString("config")

		cacheDir := getCacheDir()
		pollapoYml := loadPollapoYml(config)
		clean, _ := cmd.Flags().GetBool("clean")

		caching := cacheDeps(
			cacheDir,
			clean,
			pollapoYml,
			client,
		)

		var lockTable = PollapoRootLockTable{}
		for r := range caching {
			if r.typ == UseLockedCommitHash || r.typ == CheckCommitHash {
				lockTable[r.dep.String()] = r.revision
			}
		}

		return nil
	},
}

func cacheDeps(
	cacheDir string,
	clean bool,
	pollapoYml PollapoYml,
	client github.Client,
) chan CacheDepsEvent {
	ch := make(chan CacheDepsEvent)

	go func() {
		defer close(ch)

		if clean {
			_ = os.RemoveAll(cacheDir)
		}

		var lockTable = *pollapoYml.Root.Lock
		var queue = pollapoYml.deps()
		var visitedDeps = map[string]bool{}

		for len(queue) != 0 {
			dep := queue[0]
			queue = queue[1:]

			ymlPath := getYmlPath(cacheDir, dep)
			revType := getRevType(dep.rev)
			depString := dep.String()

			if visitedDeps[depString] == true {
				continue
			}
			visitedDeps[depString] = true

			if (revType != "branch") && (fsExists(ymlPath)) {
				queue = append(queue, loadPollapoYml(ymlPath).deps()...)
				ch <- CacheDepsEvent{
					typ:      UseCache,
					dep:      dep,
					revision: dep.rev,
				}
			}
			_ = os.MkdirAll(path.Join(cacheDir, dep.user), 0755)

			if revType == "branch" {
        commitHash := lockTable[depString]
        useLock := commitHash == ""
				if !useLock {
					commitHash = client.FetchCommitStatus(dep.user, dep.repo, dep.rev)
				}

				typ := UseLockedCommitHash
				if useLock {
          typ = UseLockedCommitHash
        } else {
          typ = CheckCommitHash
        }

        newDep := dep
        newDep.rev = commitHash
        queue = append(queue, newDep)

        // TODO create link to commit one

        ch <- CacheDepsEvent{
          typ:      typ,
          dep:      dep,
          revision: commitHash,
        }
			} else {

        // TODO download

			  ch <- CacheDepsEvent{
			    typ:      Download,
			    dep:      dep,
			    revision: dep.rev,
        }
			}
		}
	}()

	return ch
}

func getYmlPath(cacheDir string, dep PollapoDep) string { return "TODO" }
func getRevType(rev string) string {
	return "TODO"
}

func fsExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		panic(err)
	}
}

const (
	UseCache = 1 + iota
	UseLockedCommitHash
	CheckCommitHash
	Download
)

type CacheDepsEvent struct {
	typ        int
	dep      PollapoDep
	revision string
}

type PollapoDep struct {
	user string
	repo string
	rev  string
}

func (p PollapoDep) String() string {
	return fmt.Sprintf("%s/%s@%s", p.user, p.repo, p.rev)
}

func parseDep(dep string) PollapoDep {
	fmt.Println(dep)
	r := regexp.MustCompile("(.+?)/(.+?)@(.+)")
	match := r.FindStringSubmatch(dep)
	return PollapoDep{
		user: match[1],
		repo: match[2],
		rev:  match[3],
	}
}

func (p PollapoYml) deps() []PollapoDep {
	ret := make([]PollapoDep, len(*p.Deps))
	for idx, dep := range *p.Deps {
		ret[idx] = parseDep(dep)
	}
	return ret
}

type PollapoYml struct {
	Deps *[]string
	Root *PollapoRoot
}

type PollapoRoot struct {
	Lock              *PollapoRootLockTable
	ReplaceFileOption *PollapoRootReplaceFileOption `yaml:"replace-file-option"`
}

type PollapoRootLockTable map[string]string

type PollapoRootReplaceFileOption map[string]PollapoRootReplaceFileOptionItem

type PollapoRootReplaceFileOptionItem struct {
	Regex string
	Value string
}
type T struct {
	A *[]string
	B struct {
		RenamedC int   `yaml:"c"`
		D        []int `yaml:",flow"`
	}
}

func loadPollapoYml(ymlPath string) (ret PollapoYml) {
	text, _ := ioutil.ReadFile(ymlPath)

	_ = yaml.Unmarshal(text, &ret)
	return
}

func init() {
	rootCmd.AddCommand(installCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	installCmd.Flags().BoolP("clean", "c", false, "Don't use cache")
	installCmd.Flags().StringP("out-dir", "o", ".pollapo", "Out directory")
	installCmd.Flags().StringP("token", "t", "", "Github OAuth token")
	installCmd.Flags().StringP("config", "C", "./pollapo.yml", "Pollapo config")
}
