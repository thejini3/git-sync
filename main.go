package main

import (
	"encoding/json"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	hel "github.com/thejini3/go-helper"
)

const configFileName = ".config.git-sync.json"

var homeDir string
var scheduledPaths []string
var contents []theContent

func main() {
	hel.Pl("Starting git-sync")
	usr, err := user.Current()
	if err != nil {
		hel.Pl(err)
	}
	homeDir = usr.HomeDir
	hel.Pl("homeDir: " + homeDir)
	err = json.Unmarshal(hel.GetFileBytes(homeDir+"/"+configFileName), &contents)
	if err != nil {
		hel.Pl("error in json unmarshal", err)
	}
	watch()
}

func watch() {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		hel.Pl(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:

				if !ok {
					return
				}

				dir := filepath.Dir(event.Name)
				base := filepath.Base(dir)

				if content, found := getContentFromDirPath(dir); found {
					if hel.StrContains(content.IgnoreFiles, base) {
						return
					}
					execute(content)
				}

			case err, ok := <-watcher.Errors:

				if !ok {
					return
				}

				hel.Pl("watcher.Errors", "error:", err)
			}
		}
	}()

	for i := range contents {
		err = watcher.Add(contents[i].DirPath)
		if err != nil {
			hel.Pl("error in adding watcher in", contents[i], "err:", err)
		} else {
			hel.Pl("adding watcher", i+1)
			hel.PrettyPrint(&contents[i])
		}
	}

	<-done
}

func getContentFromDirPath(path string) (theContent, bool) {
	var tc theContent
	for i := range contents {
		if contents[i].DirPath == path {
			tc = contents[i]
			break
		}
	}
	return tc, len(tc.DirPath) > 0
}

func execute(c theContent) {

	if !hel.StrContains(scheduledPaths, c.DirPath) {
		hel.Pl("scheduling", c)
		scheduledPaths = append(scheduledPaths, c.DirPath)
	} else {
		return
	}

	time.AfterFunc(c.Delay*time.Second, func() {

		for _, command := range c.Commands {
			args := strings.Split(command.CommandArgs, " ")

			var cmd *exec.Cmd

			if len(args) == 0 {
				cmd = exec.Command(command.Command)
			} else {
				cmd = exec.Command(command.Command, args...)
			}

			cmd.Dir = c.DirPath

			out, err := cmd.Output()

			if err != nil {
				hel.Pl("Error running command:", command.Command, "args:", command.CommandArgs)
			}

			hel.Pl("Ran command:", command.Command, "args:", command.CommandArgs)
			hel.Pl("output:", string(out))

		}

		scheduledPaths = removeFromArray(scheduledPaths, c.DirPath)

	})

}

func removeFromArray(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
