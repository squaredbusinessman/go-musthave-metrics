package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoDirectOSExitAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noDirectOSExitAnalyzer(), "noosexitbad", "noosexitok")
}
