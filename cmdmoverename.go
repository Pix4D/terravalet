package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/dexyk/stringosim"
	"github.com/scylladb/go-set"
	"github.com/scylladb/go-set/strset"
)

func doRename(upPath, downPath, planPath, localStatePath string, fuzzyMatch bool) error {
	planFile, err := os.Open(planPath)
	if err != nil {
		return fmt.Errorf("opening the terraform plan file: %v", err)
	}
	defer planFile.Close()

	upFile, err := os.Create(upPath)
	if err != nil {
		return fmt.Errorf("creating the up file: %v", err)
	}
	defer upFile.Close()

	downFile, err := os.Create(downPath)
	if err != nil {
		return fmt.Errorf("creating the down file: %v", err)
	}
	defer downFile.Close()

	create, destroy, err := parse(planFile)
	if err != nil {
		return fmt.Errorf("parse: %v", err)
	}

	upMatches, downMatches := matchExact(create, destroy)

	msg := collectErrors(create, destroy)
	if msg != "" && !fuzzyMatch {
		return fmt.Errorf("matchExact:%v", msg)
	}

	if fuzzyMatch && create.Size() == 0 && destroy.Size() == 0 {
		return fmt.Errorf("required fuzzy-match but there is nothing left to match")
	}
	if fuzzyMatch {
		upMatches, downMatches, err = matchFuzzy(create, destroy)
		if err != nil {
			return fmt.Errorf("fuzzyMatch: %v", err)
		}
		msg := collectErrors(create, destroy)
		if msg != "" {
			return fmt.Errorf("matchFuzzy: %v", msg)
		}
	}

	stateFlags := "-state=" + localStatePath

	if err := upDownScript(upMatches, stateFlags, upFile); err != nil {
		return fmt.Errorf("writing the up script: %v", err)
	}
	if err := upDownScript(downMatches, stateFlags, downFile); err != nil {
		return fmt.Errorf("writing the down script: %v", err)
	}

	return nil
}

func doMove(upPath, downPath, srcPlanPath, dstPlanPath, srcStatePath, dstStatePath string) error {
	// We need to read srcPlanPath and dstPlanPath, while we treat as opaque
	// srcStatePath and dstStatePath
	srcPlanFile, err := os.Open(srcPlanPath)
	if err != nil {
		return fmt.Errorf("opening the terraform SRC plan file: %v", err)
	}
	defer srcPlanFile.Close()

	dstPlanFile, err := os.Open(dstPlanPath)
	if err != nil {
		return fmt.Errorf("opening the terraform DST plan file: %v", err)
	}
	defer dstPlanFile.Close()

	upFile, err := os.Create(upPath)
	if err != nil {
		return fmt.Errorf("creating the up file: %v", err)
	}
	defer upFile.Close()

	downFile, err := os.Create(downPath)
	if err != nil {
		return fmt.Errorf("creating the down file: %v", err)
	}
	defer downFile.Close()

	srcCreate, srcDestroy, err := parse(srcPlanFile)
	if err != nil {
		return fmt.Errorf("parse src-plan: %v", err)
	}
	if srcCreate.Size() > 0 {
		return fmt.Errorf("src-plan contains resources to create: %v", srcCreate.List())
	}

	dstCreate, dstDestroy, err := parse(dstPlanFile)
	if err != nil {
		return fmt.Errorf("parse dst-plan: %v", err)
	}
	if dstDestroy.Size() > 0 {
		return fmt.Errorf("dst-plan contains resources to destroy: %v", dstDestroy.List())
	}

	upMatches, downMatches := matchExact(dstCreate, srcDestroy)

	msg := collectErrors(dstCreate, srcDestroy)
	if msg != "" {
		return fmt.Errorf("matchExact:%v", msg)
	}

	upStateFlags := fmt.Sprintf("-state=%s -state-out=%s", srcStatePath, dstStatePath)
	downStateFlags := fmt.Sprintf("-state=%s -state-out=%s", dstStatePath, srcStatePath)

	if err := upDownScript(upMatches, upStateFlags, upFile); err != nil {
		return fmt.Errorf("writing the up script: %v", err)
	}
	if err := upDownScript(downMatches, downStateFlags, downFile); err != nil {
		return fmt.Errorf("writing the down script: %v", err)
	}

	return nil
}

func collectErrors(create *strset.Set, destroy *strset.Set) string {
	msg := ""
	if create.Size() != 0 {
		elems := create.List()
		sort.Strings(elems)
		msg += "\nunmatched create:\n  " + strings.Join(elems, "\n  ")
	}
	if destroy.Size() != 0 {
		elems := destroy.List()
		sort.Strings(elems)
		msg += "\nunmatched destroy:\n  " + strings.Join(elems, "\n  ")
	}
	return msg
}

// Parse the output of "terraform plan" and return two sets, the first a set of elements
// to be created and the second a set of elements to be destroyed. The two sets are
// unordered.
//
// For example:
// " # module.ci.aws_instance.docker will be destroyed"
// " # aws_instance.docker will be created"
// " # module.ci.module.workers["windows-vs2019"].aws_autoscaling_schedule.night_mode will be destroyed"
// " # module.workers["windows-vs2019"].aws_autoscaling_schedule.night_mode will be created"
func parse(rd io.Reader) (*strset.Set, *strset.Set, error) {
	var re = regexp.MustCompile(`# (.+) will be (.+)`)

	create := set.NewStringSet()
	destroy := set.NewStringSet()

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		line := scanner.Text()
		if m := re.FindStringSubmatch(line); m != nil {
			if len(m) != 3 {
				return create, destroy,
					fmt.Errorf("could not parse line %q: %q", line, m)
			}
			switch m[2] {
			case "created":
				create.Add(m[1])
			case "destroyed":
				destroy.Add(m[1])
			case "read during apply":
				// do nothing
			default:
				return create, destroy,
					fmt.Errorf("line %q, unexpected action %q", line, m[2])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return create, destroy, err
	}

	return create, destroy, nil
}

// Given two unordered sets create and destroy, perform an exact match from destroy to create.
//
// Return two maps, the first that exact matches each old element in destroy to the
// corresponding  new element in create (up), the second that matches in the opposite
// direction (down).
//
// Modify the two input sets so that they contain only the remaining (if any) unmatched elements.
//
// The criterium used to perform a matchExact is that one of the two elements must be a
// prefix of the other.
// Note that the longest element could be the old or the new one, it depends on the inputs.
func matchExact(create, destroy *strset.Set) (map[string]string, map[string]string) {
	// old -> new (or equvalenty: destroy -> create)
	upMatches := map[string]string{}
	downMatches := map[string]string{}

	// 1. Create and destroy give us the direction:
	//    terraform state mv destroy[i] create[j]
	// 2. But, for each resource, we need to know i,j so that we can match which old state
	//    we want to move to which new state, for example both are theoretically valid:
	// 	    terraform state mv module.ci.aws_instance.docker           aws_instance.docker
	//      terraform state mv           aws_instance.docker module.ci.aws_instance.docker

	for _, d := range destroy.List() {
		for _, c := range create.List() {
			if strings.HasSuffix(c, d) || strings.HasSuffix(d, c) {
				upMatches[d] = c
				downMatches[c] = d
				// Remove matched elements from the two sets.
				destroy.Remove(d)
				create.Remove(c)
			}
		}
	}

	// Now the two sets create, destroy contain only unmatched elements.
	return upMatches, downMatches
}

// Given two unordered sets create and destroy, that have already been processed by
// matchExact(), perform a fuzzy match from destroy to create.
//
// Return two maps, the first that fuzzy matches each old element in destroy to the
// corresponding  new element in create (up), the second that matches in the opposite
// direction (down).
//
// Modify the two input sets so that they contain only the remaining (if any) unmatched elements.
//
// The criterium used to perform a matchFuzzy is that one of the two elements must be a
// fuzzy match of the other, according to some definition of fuzzy.
// Note that the longest element could be the old or the new one, it depends on the inputs.
func matchFuzzy(create, destroy *strset.Set) (map[string]string, map[string]string, error) {
	// old -> new (or equvalenty: destroy -> create)
	upMatches := map[string]string{}
	downMatches := map[string]string{}

	type candidate struct {
		distance int
		create   string
		destroy  string
	}
	candidates := []candidate{}

	for _, d := range destroy.List() {
		for _, c := range create.List() {
			// Here we could also use a custom NGramSizes via
			// stringosim.QGramSimilarityOptions
			dist := stringosim.QGram([]rune(d), []rune(c))
			candidates = append(candidates, candidate{dist, c, d})
		}
	}
	sort.Slice(candidates,
		func(i, j int) bool { return candidates[i].distance < candidates[j].distance })

	for len(candidates) > 0 {
		bestCandidate := candidates[0]
		tmpCandidates := []candidate{}

		for _, c := range candidates[1:] {
			if bestCandidate.distance == c.distance {
				if (bestCandidate.create == c.create) || (bestCandidate.destroy == c.destroy) {
					return map[string]string{}, map[string]string{},
						fmt.Errorf("ambiguous migration: {%s} -> {%s} or {%s} -> {%s}",
							bestCandidate.create, bestCandidate.destroy,
							c.create, c.destroy,
						)
				}
			}
			if (bestCandidate.create != c.create) && (bestCandidate.destroy != c.destroy) {
				tmpCandidates = append(tmpCandidates, candidate{c.distance, c.create, c.destroy})
			}

		}

		candidates = tmpCandidates
		upMatches[bestCandidate.destroy] = bestCandidate.create
		downMatches[bestCandidate.create] = bestCandidate.destroy
		destroy.Remove(bestCandidate.destroy)
		create.Remove(bestCandidate.create)
	}

	return upMatches, downMatches, nil
}

// Given a map old->new, create a script that for each element in the map issues the
// command: "terraform state mv old new".
func upDownScript(matches map[string]string, stateFlags string, out io.Writer) error {
	fmt.Fprintf(out, "#! /bin/sh\n")
	fmt.Fprintf(out, "# DO NOT EDIT. Generated by terravalet.\n")
	fmt.Fprintf(out, "# terravalet_output_format=2\n")
	fmt.Fprintf(out, "#\n")
	fmt.Fprintf(out, "# This script will move %d items.\n\n", len(matches))
	fmt.Fprintf(out, "set -e\n\n")

	// -lock=false greatly speeds up operations when the state has many elements
	// and is safe as long as we use -state=FILE, since this keeps operations
	// strictly local, without considering the configured backend.
	cmd := fmt.Sprintf("terraform state mv -lock=false %s", stateFlags)

	// Go maps are unordered. We want instead a stable iteration order, to make it
	// possible to compare scripts.
	destroys := make([]string, 0, len(matches))
	for d := range matches {
		destroys = append(destroys, d)
	}
	sort.Strings(destroys)

	i := 1
	for _, d := range destroys {
		fmt.Fprintf(out, "%s \\\n    '%s' \\\n    '%s'\n\n", cmd, d, matches[d])
		i++
	}
	return nil
}
