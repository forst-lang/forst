package modulecheck

import "forst/internal/typechecker"

func revalidateDeferredWiringKeys(perPackage map[string]*typechecker.TypeChecker) error {
	for _, tc := range perPackage {
		if err := tc.RevalidateDeferredWiringKeysAfterModuleMerge(); err != nil {
			return err
		}
	}
	return nil
}
