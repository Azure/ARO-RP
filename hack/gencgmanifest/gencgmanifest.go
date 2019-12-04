//go:generate go run . -o ../../cgmanifest.json

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	outputFile = flag.String("o", "", "output file")
)

type lock struct {
	Projects []*struct {
		Name     string
		Packages []string
		Revision string
	}
}

type cgmanifest struct {
	Registrations []*registration `json:"registration,omitempty"`
	Version       int             `json:"version,omitempty"`
}

type registration struct {
	Component *typedComponent `json:"component,omitempty"`
}

type typedComponent struct {
	Type string        `json:"type,omitempty"`
	Git  *gitComponent `json:"git,omitempty"`
}

type gitComponent struct {
	CommitHash    string `json:"commitHash,omitempty"`
	RepositoryURL string `json:"repositoryUrl,omitempty"`
}

func run() error {
	var lock lock

	_, err := toml.DecodeFile("../../Gopkg.lock", &lock)
	if err != nil {
		return err
	}

	cgmanifest := &cgmanifest{
		Registrations: make([]*registration, 0, len(lock.Projects)),
		Version:       1,
	}

	for _, project := range lock.Projects {
		switch {
		case strings.HasPrefix(project.Name, "k8s.io/"):
			project.Name = strings.Replace(project.Name, "k8s.io", "github.com/kubernetes", 1)
		case strings.HasPrefix(project.Name, "cloud.google.com/") && len(project.Packages) > 0:
			project.Name += "/" + project.Packages[0]
		}

		cgmanifest.Registrations = append(cgmanifest.Registrations, &registration{
			Component: &typedComponent{
				Type: "git",
				Git: &gitComponent{
					CommitHash:    project.Revision,
					RepositoryURL: "https://" + project.Name + "/",
				},
			},
		})
	}

	b, err := json.MarshalIndent(cgmanifest, "", "    ")
	if err != nil {
		return err
	}
	b = append(b, byte('\n'))

	if *outputFile != "" {
		err = ioutil.WriteFile(*outputFile, b, 0666)
	} else {
		_, err = fmt.Print(string(b))
	}

	return err
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}
