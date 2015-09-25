package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/codegangsta/cli"
)

const (
	git = "/usr/bin/git"

	commitsShareWeight      = 0.25
	linesTouchedShareWeight = 0.35
	currentLinesShareWeight = 0.40
)

type authorStats struct {
	author            string
	commits           int
	commitsShare      float64
	linesTouched      int
	linesTouchedShare float64
	currentLines      int
	currentLinesShare float64
	suitability       float64
}

type statSet []authorStats

func (ss statSet) Len() int {
	return len(ss)
}

func (ss statSet) Less(a, b int) bool {
	return ss[a].suitability > ss[b].suitability
}

func (ss statSet) Swap(a, b int) {
	ss[a], ss[b] = ss[b], ss[a]
}

type fileStats struct {
	filename      string
	workingStats  map[string]authorStats
	finishedStats statSet
}

func getHistoricStats(folder, filename string, authorInfo map[string]authorStats) error {
	args := []string{
		"log",
		"--follow",
		"--no-merges",
		"--pretty=format:%aN",
		"--numstat",
		filename,
	}
	cmd := exec.Command(git, args...)
	cmd.Dir = folder
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	bits := strings.Split(string(out), "\n\n")
	totalCommits := 0
	totalLinesTouched := 0
	for _, b := range bits {
		lines := strings.Split(b, "\n")
		fields := strings.Fields(lines[1])
		a, err := strconv.Atoi(fields[0])
		if err != nil {
			return err
		}
		d, err := strconv.Atoi(fields[1])
		if err != nil {
			return err
		}

		if _, present := authorInfo[lines[0]]; !present {
			authorInfo[lines[0]] = authorStats{}
		}
		s := authorInfo[lines[0]]
		s.commits++
		totalCommits++
		s.linesTouched += (a + d)
		totalLinesTouched += (a + d)
		authorInfo[lines[0]] = s
	}
	for k, v := range authorInfo {
		v.commitsShare = (float64(v.commits) / float64(totalCommits)) * 100.00
		v.linesTouchedShare = (float64(v.linesTouched) / float64(totalLinesTouched)) * 100.00
		authorInfo[k] = v
	}

	return nil
}

func getCurrentStats(folder, filename string, authorInfo map[string]authorStats) error {
	args := []string{
		"blame",
		"--minimal",
		"--line-porcelain",
		filename,
	}
	cmd := exec.Command(git, args...)
	cmd.Dir = folder
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	totalLines := 0
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "author ") {
			author := l[7:]
			if _, present := authorInfo[author]; !present {
				authorInfo[author] = authorStats{}
			}
			s := authorInfo[author]
			s.currentLines++
			totalLines++
			authorInfo[author] = s
		}
	}
	for k, v := range authorInfo {
		v.currentLinesShare = (float64(v.currentLines) / float64(totalLines)) * 100.00
		authorInfo[k] = v
	}
	return nil
}

func domain(filename string) (string, error) {

	return "", nil
}

func main() {
	app := cli.NewApp()
	app.Name = "git domain"
	app.Usage = "Deduce the ownership of a file/folder"
	app.Version = "0.1.0"
	app.Author = "Roland Shoemaker"
	app.Email = "rolandshoemaker@gmail.com"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "repository-folder",
			Value: ".",
			Usage: "Folder that contains the repository (defaults to the folder you are currently in)",
		},
		cli.BoolFlag{
			Name:  "t, top",
			Usage: "Only print the name of the most suitable contributor",
		},
		cli.BoolFlag{
			Name:  "s, stripped",
			Usage: "Only print the name of the contributors (in order of suitability)",
		},
	}

	app.Action = func(c *cli.Context) {
		fs := fileStats{
			workingStats: make(map[string]authorStats),
		}

		err := getHistoricStats(c.GlobalString("repository-folder"), c.Args().First(), fs.workingStats)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = getCurrentStats(c.GlobalString("repository-folder"), c.Args().First(), fs.workingStats)
		if err != nil {
			fmt.Println(err)
			return
		}
		for k, v := range fs.workingStats {
			v.author = k
			v.suitability = (v.commitsShare * commitsShareWeight) + (v.currentLinesShare * currentLinesShareWeight) + (v.linesTouchedShare * linesTouchedShareWeight)
			fs.finishedStats = append(fs.finishedStats, v)
		}
		sort.Sort(fs.finishedStats)
		if c.GlobalBool("top") {
			fs.finishedStats = fs.finishedStats[0:1]
		}
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 2, '\t', 0)
		if !c.GlobalBool("stripped") {
			fmt.Fprintln(w, "Author\tSuitability\tTotal additions + deletions\tTotal commits\tCurrent lines")
			fmt.Fprintln(w, "------\t-----------\t---------------------------\t-------------\t-------------")
		}
		for _, v := range fs.finishedStats {
			if c.GlobalBool("stripped") {
				fmt.Fprintln(w, v.author)
			} else {
				fmt.Fprintf(w, "%s\t%.2f%%\t%d\t%d\t%d\n", v.author, v.suitability, v.linesTouched, v.commits, v.currentLines)
			}
		}
		w.Flush()
	}

	app.Run(os.Args)
}
