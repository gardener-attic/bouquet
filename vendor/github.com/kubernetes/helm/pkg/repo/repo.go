/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package repo

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"k8s.io/helm/pkg/chartutil"
)

// ChartRepository represents a chart repository
type ChartRepository struct {
	RootPath   string
	URL        string // URL of repository
	ChartPaths []string
	IndexFile  *IndexFile
}

// RepoFile represents the repositories.yaml file in $HELM_HOME
type RepoFile struct {
	Repositories map[string]string
}

// LoadRepositoriesFile takes a file at the given path and returns a RepoFile object
func LoadRepositoriesFile(path string) (*RepoFile, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var r RepoFile
	err = yaml.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// UnmarshalYAML unmarshals the repo file
func (rf *RepoFile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var repos map[string]string
	if err := unmarshal(&repos); err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			return err
		}
	}
	rf.Repositories = repos
	return nil
}

// LoadChartRepository takes in a path to a local chart repository
//      which contains packaged charts and an index.yaml file
//
// This function evaluates the contents of the directory and
// returns a ChartRepository
func LoadChartRepository(dir, url string) (*ChartRepository, error) {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	if !dirInfo.IsDir() {
		return nil, errors.New(dir + "is not a directory")
	}

	r := &ChartRepository{RootPath: dir, URL: url}

	filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			if strings.Contains(f.Name(), "index.yaml") {
				i, err := LoadIndexFile(path)
				if err != nil {
					return nil
				}
				r.IndexFile = i
			} else {
				// TODO: check for tgz extension
				r.ChartPaths = append(r.ChartPaths, path)
			}
		}
		return nil
	})

	return r, nil
}

func (r *ChartRepository) saveIndexFile() error {
	index, err := yaml.Marshal(&r.IndexFile.Entries)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(r.RootPath, indexPath), index, 0644)
}

func (r *ChartRepository) Index() error {
	if r.IndexFile == nil {
		r.IndexFile = &IndexFile{Entries: make(map[string]*ChartRef)}
	}

	for _, path := range r.ChartPaths {
		ch, err := chartutil.Load(path)
		if err != nil {
			return err
		}

		chartfile := ch.Metadata
		hash, err := generateChecksum(path)
		if err != nil {
			return err
		}

		key := chartfile.Name + "-" + chartfile.Version
		if r.IndexFile.Entries == nil {
			r.IndexFile.Entries = make(map[string]*ChartRef)
		}

		ref, ok := r.IndexFile.Entries[key]
		var created string
		if ok && ref.Created != "" {
			created = ref.Created
		} else {
			created = time.Now().UTC().String()
		}

		url, _ := url.Parse(r.URL)
		url.Path = filepath.Join(url.Path, key+".tgz")

		entry := &ChartRef{Chartfile: chartfile, Name: chartfile.Name, URL: url.String(), Created: created, Checksum: hash, Removed: false}

		r.IndexFile.Entries[key] = entry

	}

	return r.saveIndexFile()
}

func generateChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	result := sha1.Sum(b)

	return fmt.Sprintf("%x", result), nil
}
