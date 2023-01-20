package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type RepoJson struct {
	Sha string `json:"sha"`
}

type TreeJson struct {
	Tree []TreeNode `json:"tree"`
}

type TreeNode struct {
	Path string `json:"path"`
	Sha  string `json:"sha"`
	Type string `json:"type"`
	Url  string `json:"url"`
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		PrintHelp()
		os.Exit(0)
	}

	fmt.Println("> Downloading options from @github/gitignore")

	// Download options from github/gitignore
	options, err := GetOptions()
	if err != nil {
		fmt.Println("> Failed to download options")
		panic(err)
	}

	mergedContent := ""
	successes := 0

	// Download each valid arg and store content
	for _, arg := range args {
		key := strings.ToLower(arg)

		name, ok := (*options)[key]
		if !ok {
			fmt.Println("  ✖ " + key)
			continue
		}

		content, err := DownloadFile(name)
		if err != nil {
			fmt.Println("  ✖ " + arg + " (Download failed)")
		}

		mergedContent += *content + "\n"
		fmt.Println("  ✔ " + name)
		successes++
	}

	err = SaveContentToDisk(&mergedContent)
	if err != nil {
		fmt.Println("> Failed to save content to .gitignore")
		panic(err)
	}

	fmt.Printf("> Added %d entries to .gitignore", successes)
}

func PrintHelp() {
	fmt.Print("" +
		"Usage:    gitignore <lang> [...langs]\n" +
		"Example:  gitignore node sass\n")
}

func GetOptions() (*map[string]string, error) {
	var (
		res     *http.Response
		body    []byte
		repo    *RepoJson
		tree    *TreeJson
		options map[string]string
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

	options = make(map[string]string)

	for _, node := range tree.Tree {
		if strings.HasSuffix(node.Path, ".gitignore") && node.Type == "blob" {
			name := strings.TrimSuffix(node.Path, ".gitignore")
			nameLower := strings.ToLower(name)
			options[nameLower] = name
		}
	}

	return &options, nil
}

func DownloadFile(name string) (*string, error) {
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
	fo, err := os.Create(".gitignore")
	if err != nil {
		return err
	}

	defer fo.Close()

	_, err = fo.Write([]byte(*content))
	if err != nil {
		return err
	}

	return nil
}
