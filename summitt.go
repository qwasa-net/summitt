package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type patBox struct {
	regexp   *regexp.Regexp      // compiled regexp
	pattern  string              // initial re pattern (value key)
	ki       int                 // key index in the regexp
	vi       int                 // value index in the regexp
	counters map[string][2]int64 // map[key] = (total, count)
}

func main() {

	// go with errors
	var err error

	// input files
	// `-` -- stdin
	files := []string{"-"}

	// patterns
	patterns := []string{
		"^[\\-rwdx]{10} [0-9]+ [^\\s]+ [^\\s]+ *(?P<v>[0-9]+)\\s+.+(?P<k>\\.[a-z0-9]{1,4})$",
		"^\\s*(?P<v>[0-9]+)\\s+.+(?P<k>\\.[a-z0-9]{1,4})$",
		"^\\s*(?P<v>[0-9]+)\\s+(?P<k>[^\\-#\\.]+).*$",
	}

	// boxes of counters
	boxes := make([]patBox, len(patterns))

	// read patters, compile and preapare maps of counters
	for i, pat := range patterns {

		boxes[i].pattern = pat

		re, _ := regexp.Compile(pat)
		boxes[i].regexp = re

		boxes[i].counters = make(map[string][2]int64)

		// get index for named groups in the regexp
		// `k` = key, `v` = value (counter), defaults to 2 and 1
		ki := re.SubexpIndex("k")
		if ki > 0 {
			boxes[i].ki = ki
		} else {
			boxes[i].ki = 2
		}

		vi := re.SubexpIndex("v")
		if vi > 0 {
			boxes[i].vi = vi
		} else {
			boxes[i].vi = 1
		}

	}

	// count every file
	for _, filename := range files {
		err = sumFiles(filename, boxes)
		if err != nil {
			log.Fatal(err)
		}
	}

	// print counters for every box
	for i, box := range boxes {

		fmt.Printf("# #%d\n# =~ %s\n# = %d\n", (i + 1), box.pattern, len(box.counters))

		// sort by values
		type kv struct {
			k  string
			v1 int64
			v2 int64
		}
		counters := make([]kv, 0)

		for k, v := range box.counters {
			counters = append(counters, kv{k, v[0], v[1]})
		}

		_sortby := func(i, j int) bool { return counters[i].v1 < counters[j].v1 }
		sort.Slice(counters, _sortby)

		for _, c := range counters {
			fmt.Printf("%15d %7s %7d %.80s\n", c.v1, humanBytes(int64(c.v1)), c.v2, c.k)
		}

		fmt.Print("\n")

	}

}

func sumFiles(filename string, boxes []patBox) error {

	var err error

	var file *os.File

	if filename == "-" || filename == "" {
		file = os.Stdin
	} else {
		file, err = os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	scanner := bufio.NewScanner(file)

	// count lines
	for scanner.Scan() {
		line := scanner.Text()
		sumLines(line, boxes)
	}

	return err

}

func sumLines(line string, boxes []patBox) {

	for _, box := range boxes {
		matches := box.regexp.FindAllStringSubmatch(line, -1)
		if matches != nil {
			for _, match := range matches {
				if len(match) >= 3 {
					k := match[box.ki]
					k = strings.ToLower(k)
					v, _ := strconv.Atoi(match[box.vi])
					v0 := box.counters[k]
					v0[0] += int64(v)
					v0[1]++
					box.counters[k] = v0
				}
			}
		}

	}

}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d b", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}
