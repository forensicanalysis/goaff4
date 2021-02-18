<h1 align="center">goaff4</h1>

<p  align="center">
 <a href="https://codecov.io/gh/forensicanalysis/goaff4"><img src="https://codecov.io/gh/forensicanalysis/goaff4/branch/master/graph/badge.svg" alt="coverage" /></a>
 <a href="https://godocs.io/github.com/forensicanalysis/goaff4"><img src="https://godocs.io/github.com/forensicanalysis/goaff4?status.svg" alt="doc" /></a>
</p>

A Go module to read forensic disk images in the [Advanced Forensics File Format (AFF4)](http://www2.aff4.org/) as [io/fs.FS](https://golang.org/pkg/io/fs/#FS).

## Example

``` go
func main() {
	f, _ := os.Open("Base-Linear.aff4")
	info, _ := f.Stat()

	// init file system
	aff4, _ := goaff4.New(f, info.Size())

	// read root directory
	infos, _ := fs.ReadDir(aff4, ".")

	// print files
	for _, info := range infos {
		fmt.Println(info.Name())
	}
}
```
