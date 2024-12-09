//go:build ignore

package main

import (
	"bytes"
	"go/format"
	"os"
	"strings"
	"text/template"
	"unicode"
)

var TemplateTxt string = `package main

const (
{{- range .NameAndPaths}}
	{{.Name}} = "{{.Path}}"
{{- end}}
)

var SoundSrcs = []string {
{{- range .NameAndPaths}}
	"{{.Path}}",
{{- end}}
}
`

func main() {
	srcBytes, err := os.ReadFile("sound_srcs.txt")
	if err != nil {
		panic(err)
	}
	srcTxt := string(srcBytes)
	srcTxt = strings.ReplaceAll(srcTxt, "\r\n", "\n")

	type nameAndPath struct {
		Name string
		Path string
	}

	var nameAndPaths []nameAndPath

	lines := strings.Split(srcTxt, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		var name string

		for i, r := range line {
			if unicode.IsSpace(r) {
				name = line[0:i]
				line = line[i:]
				break
			}
		}

		line = strings.TrimSpace(line)
		line = strings.Trim(line, "\"'")

		var path = line

		path = strings.ReplaceAll(path, "\\", "/")

		if len(name) > 0 && len(path) > 0 {
			nameAndPaths = append(nameAndPaths, nameAndPath{
				Name: name,
				Path: path,
			})
		}
	}

	buff := &bytes.Buffer{}

	tmpl, err := template.New("sound_src_template").Parse(TemplateTxt)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(buff, struct {
		NameAndPaths []nameAndPath
	}{
		NameAndPaths: nameAndPaths,
	})
	if err != nil {
		panic(err)
	}
	formatted, err := format.Source(buff.Bytes())
	if err != nil {
		panic(err)
	}

	os.WriteFile("sound_srcs.go", formatted, 0664)
}
