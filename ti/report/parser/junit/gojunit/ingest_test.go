// Copyright 2022 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

// Copyright Josh Komoroske. All rights reserved.
// Use of this source code is governed by the MIT license,
// a copy of which can be found in the LICENSE.txt file.

package gojunit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harness/lite-engine/ti"
)

func TestExamplesInTheWild(t *testing.T) { //nolint:funlen
	tests := []struct {
		title    string
		filename string
		origin   string
		check    func(*testing.T, []Suite)
	}{
		{
			title:    "catchsoftware example",
			filename: "testdata/catchsoftware.xml",
			origin:   "https://help.catchsoftware.com/display/ET/JUnit+Format",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 2)
				assert.Len(t, suites[0].Tests, 0)
				assert.Len(t, suites[1].Tests, 3)
				assert.Equal(t, suites[1].Tests[0].Result.Desc, "Assertion failed")
				assert.Equal(t, suites[1].Tests[0].Result.Message, "test failure")
			},
		},
		{
			title:    "cubic example",
			filename: "testdata/cubic.xml",
			origin:   "https://llg.cubic.org/docs/junit/",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 1)
				assert.Equal(t, "STDOUT text", suites[0].SystemOut)
				assert.Equal(t, "STDERR text", suites[0].SystemErr)
				assert.Equal(t, "STDOUT text", suites[0].Tests[0].SystemOut)
				assert.Equal(t, "STDERR text", suites[0].Tests[0].SystemErr)
			},
		},
		{
			title:    "go-junit-report example",
			filename: "testdata/go-junit-report.xml",
			origin:   "https://github.com/jstemmer/go-junit-report/blob/master/testdata/06-report.xml",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 2)
				assert.Len(t, suites[0].Tests, 2)
				assert.Len(t, suites[1].Tests, 2)
				assert.Equal(t, "1.0", suites[0].Properties["go.version"])
				assert.Equal(t, "1.0", suites[1].Properties["go.version"])
				assert.Equal(t, suites[1].Tests[0].Result.Desc, "file_test.go:11: Error message\nfile_test.go:11: Longer\n\terror\n\tmessage.")
			},
		},
		{
			title:    "ibm example",
			filename: "testdata/ibm.xml",
			origin:   "https://www.ibm.com/support/knowledgecenter/en/SSQ2R2_14.2.0/com.ibm.rsar.analysis.codereview.cobol.doc/topics/cac_useresults_junit.html",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 1)
				assert.Equal(t, suites[0].Tests[0].Result.Desc, "\nWARNING: Use a program name that matches the source file name\nCategory: COBOL Code Review – Naming Conventions\nFile: /project/PROGRAM.cbl\nLine: 2\n      ") //nolint:lll
			},
		},
		{
			title:    "jenkinsci example",
			filename: "testdata/jenkinsci.xml",
			origin:   "https://github.com/jenkinsci/junit-plugin/blob/master/src/test/resources/hudson/tasks/junit/junit-report-1463.xml",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 6)
				assert.Equal(t, "\n", suites[0].Properties["line.separator"])
				assert.Equal(t, `\`, suites[0].Properties["file.separator"])
			},
		},
		{
			title:    "nose2 example",
			filename: "testdata/nose2.xml",
			origin:   "https://nose2.readthedocs.io/en/latest/plugins/junitxml.html",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 25)
				assert.Equal(t, suites[0].Tests[22].Result.Desc, "Traceback (most recent call last):\n  File \"nose2/tests/functional/support/scenario/tests_in_package/pkg1/test/test_things.py\", line 13, in test_typeerr\n    raise TypeError(\"oops\")\nTypeError: oops\n") //nolint:lll
			},
		},
		{
			title:    "python junit-xml example",
			filename: "testdata/python-junit-xml.xml",
			origin:   "https://pypi.org/project/junit-xml/",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 1)
				assert.Equal(t, "\n                I am stdout!\n            ", suites[0].Tests[0].SystemOut)
				assert.Equal(t, "\n                I am stderr!\n            ", suites[0].Tests[0].SystemErr)
			},
		},
		{
			title:    "surefire example",
			filename: "testdata/surefire.xml",
			origin:   "https://gist.github.com/rwbergstrom/6f0193b1a12dca9d358e6043ee6abba4",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 1)
				assert.Equal(t, "\n", suites[0].Properties["line.separator"])
				assert.Equal(t, "Hello, World\n", suites[0].Tests[0].SystemOut)
				assert.Equal(t, "I'm an error!\n", suites[0].Tests[0].SystemErr)

				var testcase = Test{
					Name:       "testStdoutStderr",
					Classname:  "com.example.FooTest",
					DurationMs: 1234560,
					Result: ti.Result{
						Status: ti.StatusFailed,
						Type:   "java.lang.AssertionError",
						Desc:   "java.lang.AssertionError\n\tat com.example.FooTest.testStdoutStderr(FooTest.java:13)\n",
					},
					Properties: map[string]string{
						"classname": "com.example.FooTest",
						"name":      "testStdoutStderr",
						"time":      "1,234.56",
					},
					SystemOut: "Hello, World\n",
					SystemErr: "I'm an error!\n",
				}

				assert.Equal(t, testcase, suites[0].Tests[0])
			},
		},
		{
			title:    "fastlane example",
			filename: "testdata/fastlane-trainer.xml",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 4)

				var testcase = Test{
					Name:       "testSomething()",
					Classname:  "TestClassSample",
					DurationMs: 342,
					Result: ti.Result{
						Status:  ti.StatusFailed,
						Message: "XCTAssertTrue failed",
						Desc:    "\n            ",
					},
					Properties: map[string]string{
						"classname": "TestClassSample",
						"name":      "testSomething()",
						"time":      "0.342",
					},
				}

				assert.Equal(t, testcase, suites[0].Tests[2])
				assert.Equal(t, suites[0].Tests[2].Result.Message, "XCTAssertTrue failed")
				assert.Equal(t, suites[0].Tests[3].Result.Type, "NullPointerException")
			},
		},
		{
			title:    "phpunit example",
			filename: "testdata/phpunit.xml",
			check: func(t *testing.T, suites []Suite) {
				assert.Len(t, suites, 1)
				assert.Len(t, suites[0].Tests, 0)
				assert.Len(t, suites[0].Suites, 1)

				suite := suites[0].Suites[0]
				assert.Len(t, suite.Tests, 1)
				assert.Len(t, suite.Suites, 2)

				assert.Equal(t, "SampleTest", suite.Name)
				assert.Equal(t, "/untitled/tests/SampleTest.php", suite.Properties["file"])

				var testcase = Test{
					Name:       "testA",
					Classname:  "SampleTest",
					DurationMs: 5,
					Result: ti.Result{
						Status: ti.StatusPassed,
					},
					Properties: map[string]string{
						"assertions": "1",
						"class":      "SampleTest",
						"classname":  "SampleTest",
						"file":       "/untitled/tests/SampleTest.php",
						"line":       "7",
						"name":       "testA",
						"time":       "0.005917",
					},
				}

				assert.Equal(t, testcase, suite.Tests[0])

				assert.Len(t, suite.Suites[1].Suites, 0)
				assert.Len(t, suite.Suites[1].Tests, 3)
				assert.Equal(t, "testC with data set #0", suite.Suites[1].Tests[0].Name)

				// checking recursive aggregation
				suites[0].Aggregate()
				actualTotals := suites[0].Totals
				expectedTotals := Totals{
					Tests:      7,
					Passed:     4,
					Skipped:    0,
					Failed:     3,
					Error:      0,
					DurationMs: 7,
				}
				assert.Equal(t, expectedTotals, actualTotals)
			},
		},
	}

	for index, test := range tests {
		name := fmt.Sprintf("#%d - %s", index+1, test.title)

		t.Run(name, func(t *testing.T) {
			suites, err := IngestFile(test.filename)
			require.NoError(t, err)
			test.check(t, suites)
		})
	}
}
