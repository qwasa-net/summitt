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
	counters map[string][3]int64 // map[key] = (total, count, max)
}

// command-line options in one structure
type flagsSet struct {
	files    []string
	patterns []string
	factor   int
	top      int
	sort     int
	reverse  bool
	lower    bool
	verbose  bool
	ignore   bool
	empty    bool
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

		boxes[i].counters = make(map[string][3]int64)

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
		err = sumFile(filename, boxes)
		if err != nil {
			fmt.Printf("#! error at file %s: %q\n", filename, err)
			if !flags.ignore {
				os.Exit(1)
			}
		}
	}

	// print counters of every box
	for i, box := range boxes {

		if len(box.counters) == 0 && flags.empty {
			continue
		}

		if flags.verbose {
			fmt.Printf("# #%d\n# =~ %s\n# × %d", (i + 1), box.pattern, len(box.counters))
			if flags.top > 0 && flags.top < len(box.counters) {
				fmt.Printf("[:%d]", flags.top)
			}
			fmt.Print("\n")
		}

		// sort by values (convert map=>slice, then sort)
		type kv struct {
			k  string
			vs [4]int64 // sum, count, ratio, max
		}

		counters := make([]kv, 0)
		for k, v := range box.counters {
			vr := v[0] / v[1]
			counters = append(counters, kv{k, [4]int64{v[0], v[1], vr, v[2]}})
		}

		// select sorder for the slice
		var sortby func(int, int) bool

		// valid options set = {1,2,3,4}
		sortMap := map[int]int{1: 0, 2: 1, 3: 2, 4: 3}
		if sortI, validSortI := sortMap[flags.sort]; validSortI {
			if flags.reverse {
				sortby = func(i, j int) bool { return counters[i].vs[sortI] > counters[j].vs[sortI] }
			} else {
				sortby = func(i, j int) bool { return counters[i].vs[sortI] < counters[j].vs[sortI] }
			}
		}

		if sortby != nil {
			sort.Slice(counters, sortby)
		}

		// cap counters slice by `top`
		if flags.top > 0 && flags.top < len(counters) {
			if flags.reverse {
				counters = counters[:flags.top]
			} else {
				counters = counters[len(counters)-flags.top:]
			}
		}

		// finally print
		for _, c := range counters {
			fmt.Printf("%15d %9s %5d %.80s\n", c.vs[0], humanBytes(int64(c.vs[0])), c.vs[1], c.k)
		}

		if flags.verbose {
			fmt.Print("\n")
		}

	}

}

// sumFile opens file and calls sumLine for every line
func sumFile(filename string, boxes []patBox) error {

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
		sumLine(line, boxes)
	}

	return err

}

// sumLine applies every pattern to the line and update counters is there is a match
func sumLine(line string, boxes []patBox) (int, error) {

	c := 0
	for _, box := range boxes {
		matches := box.regexp.FindAllStringSubmatch(line, -1)
		if matches == nil {
			continue
		}
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			k := match[box.ki]
			if flags.lower {
				k = strings.ToLower(k)
			}
			v, err := strconv.Atoi(match[box.vi])
			if err != nil && !flags.ignore {
				return c, err
			}
			v0 := box.counters[k]
			size := int64(v) * int64(flags.factor)
			v0[0] += size
			v0[1]++
			if v0[2] < size {
				v0[2] = size
			}
			box.counters[k] = v0
			c++
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
		// ls -l
		// -rw-r--r-- 1 root root     642 Mar  1  2019 passwd
		`^[\-rwxds]{10}\s+[0-9]+\s+[^\s]+\s+[^\s]+\s+(?P<v>[0-9]+)\s+.+(?P<k>\.[A-Za-z0-9]{1,4})$`,
		`^[\-rwxds]{10}\s+[0-9]+\s+[^\s]+\s+[^\s]+\s+(?P<v>[0-9]+)\s+[^\s]+\s+[^\s]+\s+[^\s]+\s+(?P<k>[^\-#\.]+).*$`,
		// ls -s
		//  4 passwd
		`^\s*(?P<v>[0-9]+)\s+.+(?P<k>\.[A-Za-z0-9]{1,4})$`,
		`^\s*(?P<v>[0-9]+)\s+(?P<k>[^\-#\.]+).*$`,
	}

	filesDefault := []string{"-"}

	var flags = flagsSet{}

	flag.BoolVar(&flags.verbose, "v", true, "= --verbose")
	flag.BoolVar(&flags.verbose, "verbose", true, "verbose output")

	flag.IntVar(&flags.sort, "s", 1, "= --sort")
	flag.IntVar(&flags.sort, "sort", 1,
		"sort by: (1) sum of counters; (2) number of entries;"+
			"(3) ratio sum/number (4) max size; (0) disable")

	flag.BoolVar(&flags.reverse, "r", false, "= --reverse")
	flag.BoolVar(&flags.reverse, "reverse", false, "reverse sorting")

	flag.BoolVar(&flags.ignore, "i", true, "= --ignore")
	flag.BoolVar(&flags.ignore, "ignore", true, "ignore errors")

	flag.BoolVar(&flags.empty, "e", true, "= --empty")
	flag.BoolVar(&flags.empty, "empty", true, "skip empty boxes")

	flag.BoolVar(&flags.lower, "l", false, "= --lower")
	flag.BoolVar(&flags.lower, "lower", false, "transform all tags to lowercase for CASE-insensitive sums")

	var k1024 bool
	flag.BoolVar(&k1024, "k", false, "= --factor=1024 (1K blocks)")
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
		fmt.Printf("# '%[1]s' calculates sums of counters for the tags,\n"+
			"e.g. total sizes for file groups from 'ls' output.\n\n"+
			"## OUTPUT FORMAT:\n  # debug info\n"+
			"    [sum] [human readable size] [number of entries] [tag]\n    …\n\n", exename)
		fmt.Printf("## USAGE:\n  $ %s [OPTIONS] [FILENAME]*\n\n", exename)
		fmt.Printf("## OPTIONS:\n")
		flag.PrintDefaults()
		if len(patternsDefault) > 0 {
			fmt.Println("\n### Default patterns (for 'ls -l' and 'ls -S' output):")
			for i, p := range patternsDefault {
				fmt.Printf("  %d: %s\n", i+1, p)
			}
		}
		fmt.Printf("\n### Example:\n  > ls -l | %[1]s\n  > ls -Rs1 | %[1]s -k\n", exename)
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

	// sort by
	if flags.sort < 0 {
		flags.sort = -flags.sort
		flags.reverse = !flags.reverse
	}

	if flags.sort < 1 || flags.sort > 4 {
		flags.sort = 0
	}

	return flags

}
