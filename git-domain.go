package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
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
	folder := "/home/roland/code/go/src/github.com/letsencrypt/boulder"
	filename := flag.String("filename", "", "")
	flag.Parse()

	fs := fileStats{
		workingStats: make(map[string]authorStats),
	}

	err := getHistoricStats(folder, *filename, fs.workingStats)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = getCurrentStats(folder, *filename, fs.workingStats)
	if err != nil {
		fmt.Println(err)
		return
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "Author\tSuitability\tTotal additions + deletions\tTotal commits\tCurrent lines")
	fmt.Fprintln(w, "------\t-----------\t---------------------------\t-------------\t-------------")
	for k, v := range fs.workingStats {
		v.author = k
		v.suitability = (v.commitsShare * commitsShareWeight) + (v.currentLinesShare * currentLinesShareWeight) + (v.linesTouchedShare * linesTouchedShareWeight)
		fs.finishedStats = append(fs.finishedStats, v)
	}
	sort.Sort(fs.finishedStats)
	for _, v := range fs.finishedStats {
		fmt.Fprintf(w, "%s\t%.2f%%\t%d\t%d\t%d\n", v.author, v.suitability, v.linesTouched, v.commits, v.currentLines)
	}
	w.Flush()
}
