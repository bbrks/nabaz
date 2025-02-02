package testengine

import (
	"errors"
	"fmt"

	"github.com/nabaz-io/nabaz/pkg/hypertest/diffengine"
	"github.com/nabaz-io/nabaz/pkg/hypertest/diffengine/parser"
	"github.com/nabaz-io/nabaz/pkg/hypertest/framework"
	"github.com/nabaz-io/nabaz/pkg/hypertest/models"
	"github.com/nabaz-io/nabaz/pkg/hypertest/scm/code"
	"github.com/nabaz-io/nabaz/pkg/hypertest/scm/history/git"
	"github.com/nabaz-io/nabaz/pkg/hypertest/storage"
	sitter "github.com/smacker/go-tree-sitter"
)

type TestEngine struct {
	LocalCode      *code.CodeDirectory
	Storage        storage.Storage
	TestFramework  framework.Framework
	LanguageParser parser.Parser
	History        git.GitHistory
	CommitId       string
	LastNabazRun   *models.NabazRun
}

func LastNabazRunResult(currentCommitId string, storage storage.Storage, gitProvider git.GitHistory) *models.NabazRun {
	for currentCommitId != "" {
		nabazResult, err := storage.NabazRunByCommitID(currentCommitId)
		if err != nil {
			panic(err)
		}
		if nabazResult != nil {
			return nabazResult
		}

		commitParents, err := gitProvider.CommitParents(currentCommitId)
		if err != nil || len(commitParents) != 1 {
			return nil
		}
		currentCommitId = commitParents[0]
	}
	return nil
}

func NewTestEngine(localCode *code.CodeDirectory, storage storage.Storage, testFramework framework.Framework,
	languageParser parser.Parser, history git.GitHistory) *TestEngine {
	commitID := history.HEAD()
	lastNabazResult := LastNabazRunResult(commitID, storage, history)
	return &TestEngine{
		LocalCode:      localCode,
		Storage:        storage,
		TestFramework:  testFramework,
		LanguageParser: languageParser,
		History:        history,
		CommitId:       commitID,
		LastNabazRun:   lastNabazResult,
	}
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func removeDuplications(s []string) []string {
	result := []string{}
	seen := make(map[string]bool)
	for _, val := range s {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = true
		}
	}
	return result
}

func (t *TestEngine) FillTestCoverageFuncNames(testRuns []models.TestRun) {
	filePathsToDictOfFuncNamesToScopes := make(map[string]map[string]*sitter.Node)

	for _, testRun := range testRuns {
		for _, scope := range testRun.CallGraph {
			fullFilePath := t.TestFramework.BasePath() + scope.Path

			funcNamesToScopes, ok := filePathsToDictOfFuncNamesToScopes[fullFilePath]
			if !ok {
				code, err := t.LocalCode.GetFileContent(fullFilePath)
				if err != nil {
					panic(fmt.Errorf("failed to get file " + fullFilePath + err.Error()))
				}

				funcNamesToScopes = t.LanguageParser.GetFunctions(code)
				filePathsToDictOfFuncNamesToScopes[fullFilePath] = funcNamesToScopes
			}

			funcName, _ := FindFunction(funcNamesToScopes, scope)
			scope.FuncName = funcName

		}

		testRun.CallGraph = removeCallGraphDups(testRun.CallGraph)
	}
}

func FindFunction(functions map[string]*sitter.Node, scope *code.Scope) (string, error) {
	for func_name, func_node := range functions {
		x1 := func_node.StartPoint().Row
		x2 := func_node.EndPoint().Row

		real_lineo := uint32(scope.StartLine - 1)
		if x1 <= real_lineo && real_lineo <= x2 {
			return func_name, nil
		}
	}

	return "", errors.New("FUNCTION NOT FOUND")
}

func removeCallGraphDups(s []*code.Scope) []*code.Scope {
	result := make([]*code.Scope, 0)
	seen := make(map[string]bool)
	for _, val := range s {
		if _, ok := seen[val.FuncName]; !ok {
			result = append(result, val)
			seen[val.FuncName] = true
		}
	}
	return result
}

func (t *TestEngine) TestsToSkip() (testsToSkip map[string]models.SkippedTest, totalTests int, err error) {
	if t.LastNabazRun != nil {
		tetsMap, err := t.TestFramework.ListTests()
		if err != nil {
			return nil, 0, err
		}
		tests := mapKeys(tetsMap)
		diffEngine := diffengine.NewDiffEngine(t.LocalCode, t.History, t.LanguageParser, t.LastNabazRun.CommitID)
		testsToSkip := t.decideWhichTestsToSkip(tests, diffEngine)
		return testsToSkip, len(tests), nil
	}

	return make(map[string]models.SkippedTest), -1, nil
}

func (engine *TestEngine) decideWhichTestsToSkip(tests []string, diffengine *diffengine.DiffEngine) map[string]models.SkippedTest {
	testsToSkip := make(map[string]models.SkippedTest)

	codeDiff, err := engine.History.Diff(engine.CommitId, engine.LastNabazRun.CommitID)
	if err != nil {
		panic(err)
	}

	changedFunctions, err := diffengine.ChangedFunctions(codeDiff)
	uniqueChangedFunctions := removeDuplications(changedFunctions)
	if err != nil {
		panic(err)
	}

	for _, test := range tests {

		skippedTest := engine.LastNabazRun.PreviousTestRun(test)
		ranTest := engine.LastNabazRun.GetTestRun(test)
		// if test is new
		if skippedTest == nil && ranTest == nil {
			continue
		}

		if skippedTest != nil {
			// test skipped in last run, should the NabazRun where it ran.
			relevantNabazResult, err := engine.Storage.NabazRunByRunID(skippedTest.RunIDRef)
			if err != nil {
				// NabazRun where it ran is not found, we should run it.
				continue
			}
			ranTest = relevantNabazResult.GetTestRun(test)
		}

		scopes := ranTest.CallGraph
		if ranTest.TestFuncScope != nil {
			scopes = append(scopes, ranTest.TestFuncScope)
		}

		// tests failed in last run or affected by changes, should run it.
		if !ranTest.Success || diffengine.Affects(uniqueChangedFunctions, scopes) {
			continue
		} else {
			// if test already skipped in last run we'll same object
			testToSkip := skippedTest
			if testToSkip == nil {
				testToSkip = &models.SkippedTest{
					Name:     ranTest.Name,
					RunIDRef: engine.LastNabazRun.RunID,
				}
			}
			testsToSkip[ranTest.Name] = *testToSkip

		}
	}
	return testsToSkip
}
