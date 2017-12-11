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
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	g2 "github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/github"
)

// The root location of the repo to start walking
var repoRoot string

// If a bulleted list should be created of GitHub names not currently collaborators
var bulletList bool

// If users should be added as pull only collaborators to the repo. Requires
// admin access for the token being used to work with the API.
var updateCollab bool

func init() {
	flag.StringVar(&repoRoot, "r", ".", "The location of the repo to start inspecting")
	flag.BoolVar(&bulletList, "b", false, "Create a bulleted list of GitHub names found for easy copy/paste")
	flag.BoolVar(&updateCollab, "a", false, "If Collaborators should be added to repo with pull only access")
}

func main() {

	flag.Parse()

	// We require a github token to make API calls due to rate limiting
	// TODO(mattfarina): There should be a better way to handle the limiting
	if os.Getenv("GITHUB_TOKEN") == "" {
		fmt.Println("Error: Please supply an environment variable named GITHUB_TOKEN with a valid token")
		os.Exit(1)
	}

	// Stores the usernames found in the OWNERS files
	handles := make(map[string]struct{})

	// Walk the file tree to gather all the GitHub Logins used in all OWNERS files
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := info.Name()
		if name == "OWNERS" {
			// found an OWNERS file. Process it.
			y, e2 := readOwners(path)
			if e2 != nil {
				return e2
			}

			for _, v := range y.Approvers {
				if _, ok := handles[v]; !ok {
					handles[v] = struct{}{}
				}
			}

			for _, v := range y.Reviewers {
				if _, ok := handles[v]; !ok {
					handles[v] = struct{}{}
				}
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error walking the directory tree:", err)
		os.Exit(1)
	}

	// Query the GitHub API for a repo to get the names of all collaborators.
	// Note, the merge functionality in prow only requires a collaborator with
	// read only access to a repo but is also in an OWNERS file.

	// Using the test-infra github client because it can handle pagenation
	// for long lists
	gc := github.NewClient(os.Getenv("GITHUB_TOKEN"), "https://api.github.com")

	users, err := gc.ListCollaborators("kubernetes", "charts")
	if err != nil {
		fmt.Println("Error getting collaborators", err)
		os.Exit(1)
	}

	// Look at all the OWNERS and find which are not collaborators
	var found bool
	var nameList []string
	for k := range handles {
		found = false
		for _, v := range users {
			if v.Login == k {
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("GitHub Login %q found in OWNERS but not a collaborator\n", k)
			nameList = append(nameList, k)
		}
	}

	if bulletList {
		fmt.Println("\nGitHub Logins as a list:")
		for _, v := range nameList {
			fmt.Println("*", v)
		}
		fmt.Println("\nFor more details on having folks become part of the k8s GitHub org see:\nhttps://github.com/kubernetes/community/blob/master/community-membership.md#requirements-for-outside-collaborators")
	}

	if updateCollab {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
		)
		tc := oauth2.NewClient(ctx, ts)

		c2 := g2.NewClient(tc)
		opts := &g2.RepositoryAddCollaboratorOptions{Permission: "pull"}
		for _, v := range nameList {
			resp, err := c2.Repositories.AddCollaborator(ctx, "kubernetes", "charts", v, opts)
			if err != nil {
				fmt.Printf("ERROR: Unable to add %q as collaborator: %s", v, err)
			} else if resp.StatusCode >= 300 || resp.StatusCode < 200 {
				fmt.Printf("ERROR: Unable to add %q as collaborator. Response %s", v, resp.Status)
			}
		}
	}

}

// The portions of the owners file we are working with
type ownersConfig struct {
	Approvers []string `json:"approvers,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
}

func readOwners(p string) (*ownersConfig, error) {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	o := &ownersConfig{}
	err = yaml.Unmarshal(b, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}
