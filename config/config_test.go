/*
Copyright 2018 The Kubernetes Authors.
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

// These tests are adapted from https://github.com/kubernetes/org/blob/main/config/config_test.go

package config

import (
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/hmarr/codeowners"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/prow/pkg/config/org"
	"sigs.k8s.io/prow/pkg/github"
)

var configPath = flag.String("config", "../peribolos.yaml", "Path to peribolos config")
var ownersDir = flag.String("owners-dir", "../", "Directory to CODEOWNERS")

var cfg org.FullConfig

func TestMain(m *testing.M) {
	flag.Parse()
	if *configPath == "" {
		fmt.Println("--config must be set")
		os.Exit(1)
	}

	if *ownersDir == "" {
		fmt.Println("--owners-dir must be set")
		os.Exit(1)
	}

	raw, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("cannot read configuation from %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		fmt.Printf("cannot unmarshal configuration from %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func loadOwners(dir string) ([]string, error) {
	var owners []string

	dir = path.Clean(dir)
	file, err := os.Open(path.Join(dir, "CODEOWNERS"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	ruleset, err := codeowners.ParseFile(file)
	if err != nil {
		return nil, err
	}

	rule, err := ruleset.Match(*configPath)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, fmt.Errorf("no matching rule found for %s", *configPath)
	}

	for _, owner := range rule.Owners {
		owners = append(owners, owner.String())
	}

	return owners, nil
}

func testDuplicates(list []string) error {
	found := sets.NewString()
	dups := sets.NewString()
	for _, i := range list {
		if found.Has(i) {
			dups.Insert(i)
		}
		found.Insert(i)
	}
	if n := len(dups); n > 0 {
		return fmt.Errorf("%d duplicate names: %s", n, strings.Join(dups.List(), ", "))
	}
	return nil
}

func isSorted(list []string) bool {
	items := make([]string, 0, len(list))
	for _, l := range list {
		items = append(items, strings.ToLower(l))
	}

	return sort.StringsAreSorted(items)
}

func normalize(s sets.Set[string]) sets.Set[string] {
	out := sets.Set[string]{}
	for _, t := range s.UnsortedList() {
		out.Insert(github.NormLogin(t))
	}
	return out
}

// testTeamMembers ensures that a user is not a maintainer and member at the same time,
// there are no duplicate names in the list, all users are org members,
// and all team repos exist in the org repos.
func testTeamMembers(teams map[string]org.Team, admins sets.Set[string], orgMembers sets.Set[string], orgRepos sets.Set[string], orgName string) []error {
	var errs []error
	for teamName, team := range teams {
		teamMaintainers := sets.New(team.Maintainers...)
		teamMembers := sets.New(team.Members...)

		teamMaintainers = normalize(teamMaintainers)
		teamMembers = normalize(teamMembers)

		// ensure all teams have privacy as closed
		if team.Privacy == nil || *team.Privacy != org.Closed {
			errs = append(errs, fmt.Errorf("The team %s in org %s doesn't have the `privacy: closed` field", teamName, orgName))
		}

		// check for non-admins in maintainers list
		if nonAdminMaintainers := teamMaintainers.Difference(admins); len(nonAdminMaintainers) > 0 {
			errs = append(errs, fmt.Errorf("The team %s in org %s has non-admins listed as maintainers; these users should be in the members list instead: %s", teamName, orgName, strings.Join(nonAdminMaintainers.UnsortedList(), ",")))
		}

		// check for users in both maintainers and members
		if both := teamMaintainers.Intersection(teamMembers); len(both) > 0 {
			errs = append(errs, fmt.Errorf("The team %s in org %s has users in both maintainer admin and member roles: %s", teamName, orgName, strings.Join(both.UnsortedList(), ", ")))
		}

		// check for duplicates
		if err := testDuplicates(team.Maintainers); err != nil {
			errs = append(errs, fmt.Errorf("The team %s in org %s has duplicate maintainers: %v", teamName, orgName, err))
		}
		if err := testDuplicates(team.Members); err != nil {
			errs = append(errs, fmt.Errorf("The team %s in org %s has duplicate members: %v", teamName, orgName, err))
		}

		// check if all are org members
		if missing := teamMembers.Difference(orgMembers); len(missing) > 0 {
			errs = append(errs, fmt.Errorf("The following members of team %s are not %s org members: %s", teamName, orgName, strings.Join(missing.UnsortedList(), ", ")))
		}
		if missing := teamMaintainers.Difference(orgMembers); len(missing) > 0 {
			errs = append(errs, fmt.Errorf("The following maintainers of team %s are not %s org members: %s", teamName, orgName, strings.Join(missing.UnsortedList(), ", ")))
		}

		// check if admins are a regular member of team
		if adminTeamMembers := teamMembers.Intersection(admins); len(adminTeamMembers) > 0 {
			errs = append(errs, fmt.Errorf("The team %s in org %s has org admins listed as members; these users should be in the maintainers list instead, and cannot be on the members list: %s", teamName, orgName, strings.Join(adminTeamMembers.UnsortedList(), ", ")))
		}

		// check if lists are sorted
		if !isSorted(team.Maintainers) {
			errs = append(errs, fmt.Errorf("The team %s in org %s has an unsorted list of maintainers", teamName, orgName))
		}
		if !isSorted(team.Members) {
			errs = append(errs, fmt.Errorf("The team %s in org %s has an unsorted list of members", teamName, orgName))
		}

		// check if team repos exist in org repos
		for repoName := range team.Repos {
			if !orgRepos.Has(repoName) {
				errs = append(errs, fmt.Errorf("The team %s in org %s references repo %s which is not defined in org repos", teamName, orgName, repoName))
			}
		}

		if team.Children != nil {
			errs = append(errs, testTeamMembers(team.Children, admins, orgMembers, orgRepos, orgName)...)
		}
	}
	return errs
}

func TestOrgs(t *testing.T) {
	own, err := loadOwners(*ownersDir)
	if err != nil {
		t.Fatalf("failed to load CODEOWNERS: %v", err)
	}

	approvers := normalize(sets.New(own...))

	for _, org := range cfg.Orgs {
		members := normalize(sets.New(org.Members...))
		admins := normalize(sets.New(org.Admins...))
		allOrgMembers := members.Union(admins)

		if diff := approvers.Difference(admins); len(diff) > 0 {
			t.Errorf("CODEOWNERS approvers who are not org admins in '%s': %s", *org.Name, strings.Join(diff.UnsortedList(), ", "))
		}

		if diff := admins.Difference(approvers); len(diff) > 0 {
			t.Errorf("org admins who are not CODEOWNERS approvers in '%s': %s", *org.Name, strings.Join(diff.UnsortedList(), ", "))
		}

		if n := len(approvers); n < 4 {
			t.Errorf("Require at least 4 approvers, found %d: %s", n, strings.Join(approvers.UnsortedList(), ", "))
		}

		if err := testDuplicates(own); err != nil {
			t.Errorf("duplicate approvers: %v", err)
		}

		if both := admins.Intersection(members); len(both) > 0 {
			t.Errorf("users in both org admin and member roles for org '%s': %s", *org.Name, strings.Join(both.UnsortedList(), ", "))
		}

		if err := testDuplicates(org.Admins); err != nil {
			t.Errorf("duplicate admins: %v", err)
		}
		allUsers := make([]string, 0, len(org.Admins)+len(org.Members))
		allUsers = append(allUsers, org.Admins...)
		allUsers = append(allUsers, org.Members...)
		if err := testDuplicates(allUsers); err != nil {
			t.Errorf("duplicate members: %v", err)
		}
		if !isSorted(org.Admins) {
			t.Errorf("admins for %s org are unsorted", *org.Name)
		}
		if !isSorted(org.Members) {
			t.Errorf("members for %s org are unsorted", *org.Name)
		}

		orgRepos := sets.New[string]()
		for repoName := range org.Repos {
			orgRepos.Insert(repoName)
		}

		if errs := testTeamMembers(org.Teams, admins, allOrgMembers, orgRepos, *org.Name); len(errs) > 0 {
			for _, err := range errs {
				t.Errorf("%v", err)
			}
		}
	}
}
