# goaff4

A Go library for the [Advanced Forensics File Format (AFF4)](http://www2.aff4.org/).

This Go library works with [io/fs](https://tip.golang.org/pkg/io/fs) which is part of Go 1.16 (Release in February 2021).

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
