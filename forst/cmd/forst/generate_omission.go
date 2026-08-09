package main

import (
	"fmt"
	"sort"
	"strings"

	transformerts "forst/internal/transformer/ts"

	"github.com/sirupsen/logrus"
)

// reportProviderOmissions logs each provider-gated omission and a summary line (F-17).
func reportProviderOmissions(outputs []*transformerts.TypeScriptOutput, log *logrus.Logger) {
	if log == nil {
		return
	}
	var omitted []transformerts.OmittedFunction
	for _, out := range outputs {
		if out == nil {
			continue
		}
		omitted = append(omitted, out.OmittedFunctions...)
	}
	if len(omitted) == 0 {
		return
	}
	sort.Slice(omitted, func(i, j int) bool {
		if omitted[i].PackageName != omitted[j].PackageName {
			return omitted[i].PackageName < omitted[j].PackageName
		}
		return omitted[i].FunctionName < omitted[j].FunctionName
	})

	var lines []string
	lines = append(lines, fmt.Sprintf("generate: omitted %d functions (unsatisfied providers)", len(omitted)))
	for _, o := range omitted {
		log.WithFields(logrus.Fields{
			"forstPackage": o.PackageName,
			"functionName": o.FunctionName,
			"reason":       o.Reason,
		}).Warnf("generate: omitted %s.%s — %s", o.PackageName, o.FunctionName, o.Reason)
		lines = append(lines, fmt.Sprintf("  %s.%s — %s", o.PackageName, o.FunctionName, o.Reason))
	}
	log.Warn(strings.Join(lines, "\n"))
}
