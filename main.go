package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"sync"
	"sync/atomic"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Count bool     `short:"n" help:"Count the number of changed repositories."`
	Paths []string `arg:"" help:"Paths to scan." default:"." type:"existingdir"`
}

var vcsPaths map[string]string
var count uint32

func main() {
	_ = kong.Parse(&CLI)

	var wg sync.WaitGroup
	wg.Add(len(CLI.Paths))
	for _, path := range CLI.Paths {
		go traverse(path, &wg)
	}
	wg.Wait()

	if CLI.Count {
		fmt.Println(count)
	}
}

func traverse(p string, wg *sync.WaitGroup) {
	defer func() { wg.Done() }()

	var args []string
	var name string
	if _, err := os.Stat(path.Join(p, ".git")); err == nil {
		args = []string{"-c", "color.status=always", "status", "-s"}
		name = "git"
	} else if _, err := os.Stat(path.Join(p, ".hg")); err == nil {
		args = []string{"--config", "extensions.color=!", "st"}
		name = "hg"
	} else if _, err := os.Stat(path.Join(p, ".svn")); err == nil {
		args = []string{"st", "-v"}
		name = "svn"
	}

	if args == nil {
		entries, err := os.ReadDir(p)
		if err != nil {
			log.Fatalln(err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				wg.Add(1)
				go traverse(path.Join(p, entry.Name()), wg)
			}
		}

		return
	}

	path, ok := vcsPaths[name]
	if !ok {
		var err error
		path, err = exec.LookPath(name)

		if err != nil {
			log.Fatalln(err)
		}
	}

	cmd := exec.Command(path, args...)
	cmd.Dir = p
	cmd.Stdin = os.Stdin

	out, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}
	if len(out) > 0 {
		if CLI.Count {
			atomic.AddUint32(&count, 1)
		} else {
			fmt.Printf("%s - %s\n%s", p, name, out)
		}
	}
}
