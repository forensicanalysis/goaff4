package goaff4

import (
	"errors"
	"io"
	"io/fs"
	"strings"

	"github.com/knakk/rdf"
)

type parsedObject struct {
	urn      string
	metadata map[string][]string
}

func (o parsedObject) add(k, v string) {
	if _, ok := o.metadata[k]; !ok {
		o.metadata[k] = []string{}
	}
	o.metadata[k] = append(o.metadata[k], v)
}

func parseObjectsFromRDF(r fs.FS) (map[string]parsedObject, error) {
	f, err := r.Open("information.turtle")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	objects := map[string]parsedObject{}

	dec := rdf.NewTripleDecoder(f, rdf.Turtle)
	for triple, err := dec.Decode(); err != io.EOF; triple, err = dec.Decode() {
		if err != nil {
			return nil, errors.New("triple is nil")
		}
		urn := triple.Subj.String()
		if _, ok := objects[urn]; !ok {
			objects[urn] = parsedObject{urn: urn, metadata: map[string][]string{}}
		}
		objects[urn].add(removeURN(triple.Pred.String()), removeURN(triple.Obj.String()))
	}
	return objects, nil
}

func removeURN(s string) string {
	if strings.Contains(s, "#") {
		p := strings.SplitN(s, "#", 2)
		return p[1]
	}
	return s
}
