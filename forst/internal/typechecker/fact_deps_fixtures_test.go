package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/testutil"
)

func assertFactDeps(fx testutil.Fixture, tc *TypeChecker) error {
	facts := tc.ActiveFactsWithDeps()
	if len(facts) == 0 {
		return fmt.Errorf("%s: expected refinement facts with deps", fx.Meta.ID)
	}
	var all []string
	for _, f := range facts {
		all = append(all, PathKeysSorted(f.Reads)...)
	}
	hasSuffix := func(suf string) bool {
		for _, k := range all {
			if strings.HasSuffix(k, suf) || k == suf || strings.Contains(k, suf) {
				return true
			}
		}
		return false
	}
	switch fx.Meta.ID {
	case "fact-deps/adult-deps-age-only":
		if !hasSuffix(".age") {
			return fmt.Errorf("want user.age dep, got %v", all)
		}
		if hasSuffix(".name") {
			return fmt.Errorf("must not include user.name, got %v", all)
		}
	case "fact-deps/valid-period-two-deps":
		if !hasSuffix(".start") || !hasSuffix(".end") {
			return fmt.Errorf("want period.start and period.end, got %v", all)
		}
	case "fact-deps/loggedin-exported-presents":
		if !hasSuffix(".session") || !hasSuffix(".user") {
			return fmt.Errorf("want session and user Present deps, got %v", all)
		}
		if len(facts) < 2 {
			return fmt.Errorf("want LoggedIn plus exported Present facts, got %d facts", len(facts))
		}
	case "fact-deps/compound-or-union-deps":
		if !hasSuffix(".age") || !hasSuffix(".role") {
			return fmt.Errorf("want age and role union deps, got %v", all)
		}
	case "fact-deps/shape-nested-email-path":
		if !hasSuffix(".user") || !hasSuffix(".email") {
			return fmt.Errorf("want nested user/email paths, got %v", all)
		}
	case "fact-deps/unanalyzable-atom-whole-root":
		foundRoot := false
		for _, f := range facts {
			for _, r := range f.Reads {
				if r != nil && len(r.Steps) == 0 {
					foundRoot = true
				}
			}
		}
		if !foundRoot {
			return fmt.Errorf("want whole-root dep, got %v", all)
		}
	case "fact-deps/allowedfor-cross-value-deps":
		if !hasSuffix(".amount") || !hasSuffix(".balance") || !hasSuffix(".status") {
			return fmt.Errorf("want withdrawal.amount + account.balance/status, got %v", all)
		}
	case "fact-deps/nested-guard-cached-body":
		if !hasSuffix(".age") {
			return fmt.Errorf("Grown should include Adult's user.age, got %v", all)
		}
	case "fact-deps/binary-ensure-age-path":
		if !hasSuffix(".age") {
			return fmt.Errorf("want user.age path, got %v", all)
		}
	case "fact-deps/type-target-deps-place":
		found := false
		for _, f := range facts {
			if f.Subject != nil {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("want type-target fact on status place")
		}
	}
	return nil
}
