package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Count bool     `short:"n" help:"Count the number of changed repositories."`
	Paths []string `arg:"" help:"Paths to scan." default:"." type:"existingdir"`
}

var gitPath string
var hgPath string
var svnPath string

type VCSKind uint8

const (
	NONE = iota
	GIT
	HG
	SVN
)

func main() {
	_ = kong.Parse(&CLI)

	workers := len(CLI.Paths)
	// we're using this homegrown version of sync.WaitGroup so that we can wait
	// for changes to the worker count while also waiting for stdout output
	wg := make(chan int)

	if CLI.Count {
		countSoFar := 0
		count := make(chan struct{})
		for _, path := range CLI.Paths {
			go traverseCount(path, wg, count)
		}

		for {
			select {
			case _, ok := <-count:
				if !ok {
					fmt.Println(countSoFar)
					os.Exit(0)
				}

				countSoFar += 1
			case ch, ok := <-wg:
				if !ok {
					continue
				}

				workers += ch
				if workers == 0 {
					close(count)
				}
			}
		}
	} else {
		vcsOut := make(chan []byte)
		for _, path := range CLI.Paths {
			go traverse(path, wg, vcsOut)
		}

		firstOut := true

		for {
			select {
			case out, ok := <-vcsOut:
				if !ok {
					os.Exit(0)
				}

				if firstOut {
					firstOut = false
				} else {
					_, err := os.Stdout.Write([]byte{byte('\n')})
					if err != nil {
						log.Fatalln(err)
					}
				}
				_, err := os.Stdout.Write(out)
				if err != nil {
					log.Fatalln(err)
				}
			case ch, ok := <-wg:
				if !ok {
					continue
				}

				workers += ch
				if workers == 0 {
					close(vcsOut)
				}
			}
		}
	}
}

func traverseCount(p string, wg chan int, count chan struct{}) {
	defer func() { wg <- -1 }()

	var kind VCSKind = NONE
	if _, err := os.Stat(path.Join(p, ".git")); err == nil {
		kind = GIT
	} else if _, err := os.Stat(path.Join(p, ".hg")); err == nil {
		kind = HG
	} else if _, err := os.Stat(path.Join(p, ".svn")); err == nil {
		kind = SVN
	}

	if kind != NONE {
		err := find(kind)
		if err != nil {
			log.Fatalln(err)
		}
		var cmd *exec.Cmd
		switch kind {
		case GIT:
			cmd = exec.Command(gitPath, "-c", "color.status=always", "status", "-s")
		case HG:
			cmd = exec.Command(hgPath, "--config", "extensions.color=!", "st")
		case SVN:
			cmd = exec.Command(svnPath, "st", "-v")
		}
		cmd.Dir = p
		cmd.Stdin = os.Stdin
		out, err := cmd.Output()
		if err != nil {
			log.Fatalln(err)
		}
		if len(out) > 0 {
			count <- struct{}{}
		}
	} else {
		entries, err := os.ReadDir(p)
		if err != nil {
			log.Fatalln(err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				wg <- 1
				go traverseCount(path.Join(p, entry.Name()), wg, count)
			}
		}

	}
}

func traverse(p string, wg chan int, vcsOut chan []byte) {
	defer func() { wg <- -1 }()

	var kind VCSKind = NONE
	if _, err := os.Stat(path.Join(p, ".git")); err == nil {
		kind = GIT
	} else if _, err := os.Stat(path.Join(p, ".hg")); err == nil {
		kind = HG
	} else if _, err := os.Stat(path.Join(p, ".svn")); err == nil {
		kind = SVN
	}

	if kind != NONE {
		err := find(kind)
		if err != nil {
			log.Fatalln(err)
		}
		var cmd *exec.Cmd
		var name string
		switch kind {
		case GIT:
			cmd = exec.Command(gitPath, "-c", "color.status=always", "status", "-s")
			name = "git"
		case HG:
			cmd = exec.Command(hgPath, "--config", "extensions.color=!", "st")
			name = "hg"
		case SVN:
			cmd = exec.Command(svnPath, "st", "-v")
			name = "svn"
		}
		cmd.Dir = p
		cmd.Stdin = os.Stdin
		var res []byte = []byte(fmt.Sprintf("%s - %s\n", p, name))
		out, err := cmd.Output()
		if err != nil {
			log.Fatalln(err)
		}
		if len(out) > 0 {
			vcsOut <- append(res, out...)
		}
	} else {
		entries, err := os.ReadDir(p)
		if err != nil {
			log.Fatalln(err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				wg <- 1
				go traverse(path.Join(p, entry.Name()), wg, vcsOut)
			}
		}

	}
}

func find(kind VCSKind) error {
	var err error
	switch kind {
	case GIT:
		if gitPath == "" {
			gitPath, err = exec.LookPath("git")
		}
	case HG:
		if hgPath == "" {
			hgPath, err = exec.LookPath("hg")
		}
	case SVN:
		if svnPath == "" {
			svnPath, err = exec.LookPath("svn")
		}
	}
	return err
}
