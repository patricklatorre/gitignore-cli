package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

const Version = "0.1.2"

type RepoJson struct {
	Sha string `json:"sha"`
}

type RepoTreeJson struct {
	Tree []RepoNodeJson `json:"tree"`
}

type RepoNodeJson struct {
	Path string `json:"path"`
	Sha  string `json:"sha"`
	Type string `json:"type"`
	Url  string `json:"url"`
}

func main() {
	// Parse args
	flag.Usage = PrintUsageAndExit
	flagHelp := flag.Bool("h", false, "Print the help screen")
	flagVersion := flag.Bool("v", false, "Print the gitignore version")
	flag.Parse()

	args := flag.Args()

	// Handle flags
	if *flagVersion {
		PrintVersionAndExit()
	} else if *flagHelp || len(args) == 0 {
		PrintUsageAndExit()
	}

	fmt.Println("> Downloading choices from @github/gitignore")

	// Download choices from github/gitignore
	choices, err := GetChoices()
	if err != nil {
		fmt.Println("> Failed to download choices")
		panic(err)
	}

	var (
		wg sync.WaitGroup

		mergedContentMu sync.Mutex
		mergedContent   string

		totalDoneMu sync.Mutex
		totalDone   int
	)

	// Add each arg as a job
	wg.Add(len(args))

	// Download each valid arg
	for _, arg := range args {
		arg := arg
		key := strings.ToLower(arg)

		choiceName, ok := (*choices)[key]
		if !ok {
			fmt.Println("  ✖ " + key)
			wg.Done()
			continue
		}

		// Download concurrently
		go func() {
			choiceContent, err := DownloadChoice(choiceName)
			if err != nil {
				fmt.Println("  ✖ " + arg + " (Download failed)")
				wg.Done()
			}

			mergedContentMu.Lock()
			mergedContent += *choiceContent + "\n"
			mergedContentMu.Unlock()

			totalDoneMu.Lock()
			totalDone++
			totalDoneMu.Unlock()

			fmt.Println("  ✔ " + choiceName)
			wg.Done()
		}()
	}

	wg.Wait()

	err = SaveContentToDisk(&mergedContent)
	if err != nil {
		fmt.Println("> Failed to save content to .gitignore")
		panic(err)
	}

	fmt.Printf("> Added %d entries to .gitignore\n", totalDone)
}

func GetChoices() (*map[string]string, error) {
	var (
		res     *http.Response
		body    []byte
		repo    *RepoJson
		tree    *RepoTreeJson
		choices map[string]string
		err     error
	)

	res, err = http.Get("https://api.github.com/repos/github/gitignore/commits/main")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &repo)
	if err != nil {
		return nil, err
	}

	res, err = http.Get("https://api.github.com/repos/github/gitignore/git/trees/" + repo.Sha)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &tree)
	if err != nil {
		return nil, err
	}

	choices = make(map[string]string)

	for _, node := range tree.Tree {
		if strings.HasSuffix(node.Path, ".gitignore") && node.Type == "blob" {
			name := strings.TrimSuffix(node.Path, ".gitignore")
			nameLower := strings.ToLower(name)
			choices[nameLower] = name
		}
	}

	return &choices, nil
}

func DownloadChoice(name string) (*string, error) {
	var (
		res  *http.Response
		body []byte
		data string
		err  error
	)

	res, err = http.Get(fmt.Sprintf("https://raw.githubusercontent.com/github/gitignore/main/%s.gitignore", name))
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	data = string(body)

	return &data, nil
}

func SaveContentToDisk(content *string) error {
	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(*content)
	if err != nil {
		return err
	}

	return nil
}

func PrintUsageAndExit() {
	fmt.Print("" +
		"Usage:    gitignore <lang> [...langs]\n" +
		"Example:  gitignore node sass\n\n")

	flag.PrintDefaults()
	os.Exit(0)
}

func PrintVersionAndExit() {
	fmt.Printf("Gitignore CLI %s\n", Version)
	os.Exit(0)
}
