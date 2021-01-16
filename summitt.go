package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

// command-line options in one structure
type flagsSet struct {
	files    []string
	patterns []string
	factor   int
	top      int
	reverse  bool
	lower    bool
	verbose  bool
	ignore   bool
}

var flags flagsSet

func main() {

	// go with errors
	var err error

	flags = readFlags()

	// build boxes for counters
	boxes := make([]patBox, len(flags.patterns))

	// read patterns, compile and prepare maps of counters
	for i, pat := range flags.patterns {

		boxes[i].pattern = pat

		re, err := regexp.Compile(pat)
		if err != nil {
			fmt.Printf("#! pattern error `%q`: %q\n", pat, err)
			if !flags.ignore {
				os.Exit(1)
			}
		}

		boxes[i].regexp = re

		boxes[i].counters = make(map[string][2]int64)

		// get index for the named groups in the regexp
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
	for _, filename := range flags.files {
		err = sumFiles(filename, boxes)
		if err != nil {
			fmt.Printf("#! error at file %s: %q\n", filename, err)
			if !flags.ignore {
				os.Exit(1)
			}
		}
	}

	// print counters of every box
	for i, box := range boxes {

		if flags.verbose {
			fmt.Printf("# #%d\n# =~ %s\n# = %d\n", (i + 1), box.pattern, len(box.counters))
		}

		// sort by values (convert map=>slice, then sort)
		type kv struct {
			k  string
			v1 int64
			v2 int64
		}

		counters := make([]kv, 0)
		for k, v := range box.counters {
			counters = append(counters, kv{k, v[0], v[1]})
		}

		var _sortby func(i, j int) bool
		if flags.reverse {
			_sortby = func(i, j int) bool { return counters[i].v1 > counters[j].v1 }
		} else {
			_sortby = func(i, j int) bool { return counters[i].v1 < counters[j].v1 }
		}

		sort.Slice(counters, _sortby)

		// finally print
		for i, c := range counters {
			if flags.top > 0 && (i+1) > flags.top {
				break
			}
			fmt.Printf("%15d %9s %5d %.80s\n", c.v1, humanBytes(int64(c.v1)), c.v2, c.k)
		}

		if flags.verbose {
			fmt.Print("\n")
		}

	}

}

// sumFiles opens file and calls sumLines for every line
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

// sumLines applies every pattern to the line and update counters is there is a match
func sumLines(line string, boxes []patBox) (int, error) {

	c := 0
	for _, box := range boxes {
		matches := box.regexp.FindAllStringSubmatch(line, -1)
		if matches != nil {
			for _, match := range matches {
				if len(match) >= 3 {
					k := match[box.ki]
					if flags.lower {
						k = strings.ToLower(k)
					}
					v, err := strconv.Atoi(match[box.vi])
					if err != nil && !flags.ignore {
						return c, err
					}
					v0 := box.counters[k]
					v0[0] += int64(v) * int64(flags.factor)
					v0[1]++
					box.counters[k] = v0
					c++
				}
			}
		}

	}

	return c, nil

}

// humanBytes was shamelessly stolen from somewhere
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%db", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}

type flagsArray []string

func (i *flagsArray) String() string {
	return ""
}

func (i *flagsArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func readFlags() flagsSet {

	patternsDefault := []string{
		`^[\-rwxds]{10}\s+[0-9]+\s+[^\s]+\s+[^\s]+\s+(?P<v>[0-9]+)\s+.+(?P<k>\.[A-Za-z0-9]{1,4})$`,
		`^\s*(?P<v>[0-9]+)\s+.+(?P<k>\.[A-Za-z0-9]{1,4})$`,
		`^\s*(?P<v>[0-9]+)\s+(?P<k>[^\-#\.]+).*$`,
	}

	filesDefault := []string{"-"}

	var flags = flagsSet{}

	flag.BoolVar(&flags.verbose, "v", true, "= --verbose")
	flag.BoolVar(&flags.verbose, "verbose", true, "verbose output")

	flag.BoolVar(&flags.reverse, "r", false, "= --reverse")
	flag.BoolVar(&flags.reverse, "reverse", false, "reverse sorting")

	flag.BoolVar(&flags.ignore, "i", true, "= --ignore")
	flag.BoolVar(&flags.ignore, "ignore", true, "ignore errors")

	flag.BoolVar(&flags.lower, "l", false, "= --lower")
	flag.BoolVar(&flags.lower, "lower", false, "transform all tags to lowercase for CASE-insensitive sums")

	var k1024 bool
	flag.BoolVar(&k1024, "k", false, "(1K blocks) = --factor=1024")
	flag.IntVar(&flags.factor, "factor", 1, "counter factor ×F")

	flag.IntVar(&flags.top, "t", -1, "= --top")
	flag.IntVar(&flags.top, "top", -1, "N top lines")

	var patterns flagsArray
	flag.Var(&patterns, "p", "= --pattern")
	flag.Var(&patterns, "pattern", "parsing pattern")

	var files flagsArray
	flag.Var(&files, "f", "inputfile (default stdin)")

	usage := func() {
		exename := filepath.Base(os.Args[0])
		fmt.Printf("'%s' calculates sums of counters for the tags,\n"+
			"e.g. total sizes for file groups from 'ls' output. \n\n"+
			"OUTPUT: [sum] [human readable size] [number of occurrences] [tag] \n\n", exename)
		fmt.Printf("Usage: %s [OPTIONS] filename ...\n\n", exename)
		flag.PrintDefaults()
		if len(patternsDefault) > 0 {
			fmt.Println("\nDefault patterns (for 'ls -l' and 'ls -S' output):")
			for i, p := range patternsDefault {
				fmt.Printf("%d: %s\n", i+1, p)
			}
		}
		fmt.Printf("\nExample:\n > ls -l | %[1]s\n > ls -Rs1 | %[1]s -k\n", exename)
	}

	flag.Usage = usage

	flag.Parse()

	// also get filenames from non-flags
	for _, filename := range flag.Args() {
		files.Set(filename)
	}

	// list of files -- stdin by default
	if len(files) > 0 {
		flags.files = files
	} else {
		flags.files = filesDefault
	}

	// list of patterns
	if len(patterns) > 0 {
		flags.patterns = patterns
	} else {
		flags.patterns = patternsDefault
	}

	// 1K blocks shortcut
	if k1024 {
		flags.factor = 1024
	}

	return flags

}
