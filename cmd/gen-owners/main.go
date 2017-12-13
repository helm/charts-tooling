/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

// This application reads in the Chart.yaml file and tries to generate an
// OWNERS file

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"k8s.io/helm/pkg/chartutil"
)

// The chart file named location
var chartYaml string

// If an OWNERS file should be written to disk next to the Chart.yaml file
var writeOwners bool

// If the helmignore should have OWNERS appended to it
var updateHelmIgnore bool

// If bitnami-bot is found in a list add real people from Bitnami as well
var handleBitnami bool

func init() {
	flag.StringVar(&chartYaml, "c", "Chart.yaml", "Location of the Chart.yaml file")
	flag.BoolVar(&writeOwners, "o", false, "If the OWNERS file should be written")
	flag.BoolVar(&updateHelmIgnore, "i", false, "If the OWNERS file should be appended to .helmignore")
	flag.BoolVar(&handleBitnami, "b", false, "Handle the bitnami-bot by adding real people at Bitnami")
}

func main() {

	flag.Parse()

	// Read Charts.yaml file
	chart, err := chartutil.LoadChartfile(chartYaml)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ownrs := &ownersConfig{}

	// Interate over the maintainers
	for _, maintainer := range chart.Maintainers {

		// Check if user name has no spaces (spaces mean it's not a username)
		if !strings.Contains(maintainer.Name, " ") {
			// If there is no space see if the username is valid
			resp, err := http.Get("https://github.com/" + maintainer.Name)
			if err != nil {
				fmt.Println("Error: Problem fetching", maintainer.Name, err)
			} else {

				// Don't need the content
				resp.Body.Close()

				if resp.StatusCode != 200 {

					// TODO(mattfarina): try doing a search
					fmt.Println("Error: Problem fetching", maintainer.Name, err)

					n := lookupName(maintainer.Name, maintainer.Email)
					if n != "" {
						ownrs.Approvers = append(ownrs.Approvers, n)
						ownrs.Reviewers = append(ownrs.Reviewers, n)
					}
				} else {
					// We found a valid name
					ownrs.Approvers = append(ownrs.Approvers, maintainer.Name)
					ownrs.Reviewers = append(ownrs.Reviewers, maintainer.Name)
				}
			}
		} else {
			// There's a space which means a name
			n := lookupName(maintainer.Name, maintainer.Email)
			if n != "" {
				ownrs.Approvers = append(ownrs.Approvers, n)
				ownrs.Reviewers = append(ownrs.Reviewers, n)
			}
		}
	}

	if handleBitnami {
		for _, v := range ownrs.Approvers {
			if v == "bitnami-bot" {
				ownrs.Approvers = append(ownrs.Approvers, "prydonius", "tompizmor", "sameersbn")
			}
		}

		for _, v := range ownrs.Reviewers {
			if v == "bitnami-bot" {
				ownrs.Reviewers = append(ownrs.Reviewers, "prydonius", "tompizmor", "sameersbn")
			}
		}
	}

	out, err := yaml.Marshal(ownrs)
	if err != nil {
		fmt.Println("yaml.Marshal error", err)
		os.Exit(1)
	}
	fmt.Println("OWNERS file content:")
	fmt.Println(string(out))

	d := filepath.Dir(chartYaml)

	if writeOwners {
		fmt.Println("Writing owners file")
		err = ioutil.WriteFile(filepath.Join(d, "OWNERS"), out, 0644)
		if err != nil {
			fmt.Println("Error writing owners file:", err)
			os.Exit(1)
		}
	}

	if updateHelmIgnore {
		fmt.Println("Appending OWNERS to .helmignore")
		hp := filepath.Join(d, ".helmignore")
		f, err := os.OpenFile(hp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error appending .helmignore file:", err)
			os.Exit(1)
		}
		if _, err := f.Write([]byte("# OWNERS file for Kubernetes\nOWNERS\n")); err != nil {
			fmt.Println("Error appending .helmignore file:", err)
			os.Exit(1)
		}
		if err := f.Close(); err != nil {
			fmt.Println("Error appending .helmignore file:", err)
			os.Exit(1)
		}
	}
}

// The portions of the owners file we are working with
type ownersConfig struct {
	Approvers []string `json:"approvers,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
}

func lookupName(name, email string) string {

	// We require a github token to make API calls due to rate limiting
	// TODO(mattfarina): There should be a better way to handle the limiting
	if os.Getenv("GITHUB_TOKEN") == "" {
		fmt.Println("Error: Please supply an environment variable named GITHUB_TOKEN with a valid token")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// Do a user search
	opts := &github.SearchOptions{Sort: "created", Order: "asc"}
	users, _, err := client.Search.Users(ctx, name+" "+email, opts)

	if err != nil {
		fmt.Println("Unable to search of name", name, err)
		return ""
	}

	if len(users.Users) == 1 {
		for _, v := range users.Users {
			fmt.Printf("Found github id %q for name %q\n", v.GetLogin(), name)
			return v.GetLogin()
		}

	} else if len(users.Users) > 1 {
		fmt.Println("WARNING: Found multiple names for", name, "please try to manually find the names")
		return ""
	}

	fmt.Println("WARNING: Unable to file a username for", name)
	return ""
}
