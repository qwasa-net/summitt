# summitt

![Go](https://github.com/qwasa-net/summitt/workflows/Go/badge.svg)

'summitt' calculates sums of counters for the tags,
e.g. total sizes for file groups from 'ls' output.

```shell
# 'summitt' calculates sums of counters for the tags,
e.g. total sizes for file groups from 'ls' output.

## OUTPUT FORMAT:
  # debug info
    [sum] [human readable size] [number of occurrences] [tag]
    …

## USAGE:
  $ summitt [OPTIONS] [FILENAME]*

## OPTIONS:
  -e    = --empty (default true)
  -empty
        skip empty boxes (default true)
  -f value
        inputfile (default stdin)
  -factor int
        counter factor ×F (default 1)
  -i    = --ignore (default true)
  -ignore
        ignore errors (default true)
  -k    = --factor=1024 (1K blocks)
  -l    = --lower
  -lower
        transform all tags to lowercase for CASE-insensitive sums
  -p value
        = --pattern
  -pattern value
        parsing pattern
  -r    = --reverse
  -reverse
        reverse sorting
  -s int
        = --sort (default 1)
  -sort int
        sort by: (1) counters sum; (2) number of occurrences; (0) disable (default 1)
  -t int
        = --top (default -1)
  -top int
        N top lines (default -1)
  -v    = --verbose (default true)
  -verbose
        verbose output (default true)

### Default patterns (for 'ls -l' and 'ls -S' output):
  1: ^[\-rwxds]{10}\s+[0-9]+\s+[^\s]+\s+[^\s]+\s+(?P<v>[0-9]+)\s+.+(?P<k>\.[A-Za-z0-9]{1,4})$
  2: ^\s*(?P<v>[0-9]+)\s+.+(?P<k>\.[A-Za-z0-9]{1,4})$
  3: ^\s*(?P<v>[0-9]+)\s+(?P<k>[^\-#\.]+).*$

### Example:
  > ls -l | summitt
  > ls -Rs1 | summitt -k
```


---------------------------------------

```shell
user@host ~> ls -l | summitt -l
# #1
# =~ ^[\-rwxds]{10}\s+[0-9]+\s+[^\s]+\s+[^\s]+\s+(?P<v>[0-9]+)\s+.+(?P<k>\.[A-Za-z0-9]{1,4})$
# × 3
    40891959871    38.1GB    36 .log
    46151918113    43.0GB    36 .sql
    83009816229    77.3GB    72 .bz2

# #2
# =~ ^[\-rwxds]{10}\s+[0-9]+\s+[^\s]+\s+[^\s]+\s+(?P<v>[0-9]+)\s+[^\s]+\s+[^\s]+\s+[^\s]+\s+(?P<k>[^\-#\.]+).*$
# × 3
    46209682030    43.0GB    48 errors
    57938769361    54.0GB    48 database_dump
    65905242822    61.4GB    48 report
```