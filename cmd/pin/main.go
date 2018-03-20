package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pinboardTag struct {
	Name     string
	UseCount uint64
}

type alphaAsc []pinboardTag
type alphaDsc []pinboardTag
type useAsc []pinboardTag
type useDsc []pinboardTag

func (x alphaAsc) Len() int           { return len(x) }
func (x alphaAsc) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x alphaAsc) Less(i, j int) bool { return x[i].Name < x[j].Name }

func (x alphaDsc) Len() int           { return len(x) }
func (x alphaDsc) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x alphaDsc) Less(i, j int) bool { return x[i].Name > x[j].Name }

func (x useAsc) Len() int           { return len(x) }
func (x useAsc) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x useAsc) Less(i, j int) bool { return x[i].UseCount < x[j].UseCount }

func (x useDsc) Len() int           { return len(x) }
func (x useDsc) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x useDsc) Less(i, j int) bool { return x[i].UseCount > x[j].UseCount }

func getTags(cmd *cobra.Command, args []string) error {

	alpha, err := cmd.Flags().GetBool("alphabetical")
	if err != nil {
		return err
	}
	desc, err := cmd.Flags().GetBool("descending")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.pinboard.in/v1/tags/get?auth_token=%s&format=json", cmd.Flag("token").Value)
	log.Debug(fmt.Sprintf("GET %s...", url))
	rsp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	log.Debug(fmt.Sprintf("GET %s...done(%d).", url, rsp.StatusCode))

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	if rsp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	var tags map[string]string
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return err
	}

	tagsSlice := make([]pinboardTag, len(tags))
	idx := 0
	maxTagLen := 0
	maxUseCount := uint64(0)
	for k, v := range tags {
		if len(k) > maxTagLen {
			maxTagLen = len(k)
		}
		uc, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return err
		}
		if uc > maxUseCount {
			maxUseCount = uc
		}
		tagsSlice[idx] = pinboardTag{Name: k, UseCount: uc}
		idx += 1
	}
	maxUseCount = uint64(math.Log10(float64(maxUseCount))) + 1

	if alpha {
		if desc {
			sort.Sort(alphaDsc(tagsSlice))
		} else {
			sort.Sort(alphaAsc(tagsSlice))
		}
	} else {
		if desc {
			sort.Sort(useDsc(tagsSlice))
		} else {
			sort.Sort(useAsc(tagsSlice))
		}
	}

	if maxUseCount < 9 {
		maxUseCount = 9 // len("Use Count")
	}
	format := fmt.Sprintf("| %%-%ds | %%%dd |\n", maxTagLen, maxUseCount)
	fmt.Printf(fmt.Sprintf("| %%-%ds | %%%ds |\n", maxTagLen, maxUseCount), "Tag", "Use Count")
	rule := fmt.Sprintf("+%s+%s+", strings.Repeat("-", int(maxTagLen+2)), strings.Repeat("-", int(maxUseCount+2)))
	fmt.Println(rule)
	for i := 0; i < len(tagsSlice); i++ {
		k := tagsSlice[i].Name
		v := tagsSlice[i].UseCount
		fmt.Printf(format, k, v)
	}
	fmt.Println(rule)

	return nil
}

func renameTags(cmd *cobra.Command, args []string) error {

	old := args[0]
	new := args[1]

	url := fmt.Sprintf("https://api.pinboard.in/v1/tags/rename?auth_token=%s&old=%s&new=%s&format=json", cmd.Flag("token").Value, old, new)
	log.Debug(fmt.Sprintf("GET %s...", url))
	rsp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	log.Debug(fmt.Sprintf("GET %s...done(%d).", url, rsp.StatusCode))

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	if rsp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	fmt.Printf("%v\n", body)
	return nil
}

var getTagsCmd = &cobra.Command{
	Use:   "get-tags",
	Short: "Retrieve all your tags along with their use counts",
	RunE:  getTags,
}

var renameTagsCmd = &cobra.Command{
	Use:   "rename-tags [old] [new]",
	Short: "Rename a tag, or fold it into an existing tag",
	Args:  cobra.ExactArgs(2),
	RunE:  renameTags,
}

func init() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	getTagsCmd.Flags().BoolP("alphabetical", "a", false, "Sort alphabetically")
	getTagsCmd.Flags().BoolP("descending", "d", false, "Sort in descending order")
}

func main() {

	// TODO(sp1ff): Add --version flag
	var rootCmd = &cobra.Command{
		Use:           "app",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// TODO(sp1ff): Come up with other ways to specify (~/.pin, environment, e.g.)
	rootCmd.PersistentFlags().StringP("token", "t", "", "Your pinboard.in API token (required)")
	rootCmd.MarkFlagRequired("token")
	rootCmd.AddCommand(getTagsCmd, renameTagsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
