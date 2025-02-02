#! /usr/bin/python3
from ast import arg
import contextlib
import json
import sys
import time
from collections import defaultdict
from pathlib import Path
from typing import List, Optional

import pytest
from coverage import CoverageData
from pydantic import BaseModel
from pytest_jsonreport.plugin import JSONReport


class HashableBaseModel(BaseModel):
    def __hash__(self):  # make hashable BaseModel subclass
        return hash((type(self),) + tuple(self.__dict__.values()))


class Scope(HashableBaseModel):
    path: str = None
    func_name: str = None
    file: str = None
    startline: int = None
    startcol: int = None
    endline: int = None
    endcol: int = None


class TestResult(HashableBaseModel):
    call_graph: List[Scope]
    test_func_scope: Optional[Scope]
    name: str
    success: bool
    time_in_ms: float


class TestSelectionPlugin:
    def __init__(self, tests_to_skip):
        self.tests_to_skip = tests_to_skip

    def pytest_collection_modifyitems(self, config, items):
        xfail_mark = pytest.mark.xfail(run=False, reason="skipped by nabaz.io")
        skip_mark = pytest.mark.skip(reason="skipped by nabaz.io")

        for test in items:
            test_to_skip = self.tests_to_skip.get(test.nodeid)
            if test_to_skip != None:
                if test_to_skip == True:  # passed == true
                    test.add_marker(skip_mark)
                else:
                    test.add_marker(xfail_mark)


def run_tests(tests_to_skip, args):
    json_report = JSONReport()

    run_tests_start_time = time.time()
    status_code = pytest.main(
        args,
        plugins=[TestSelectionPlugin(tests_to_skip), json_report],
    )
    run_tests_time = time.time() - run_tests_start_time
    coverage_data = _get_coverage_data()

    ran_tests = [
        TestResult(name=test["nodeid"],
                   success=is_passed(test["outcome"]),
                   time_in_ms=calculate_test_time(test),
                   call_graph=list(coverage_data.get(test["nodeid"], set({}))))
        for test in json_report.report["tests"] if not is_skipped(test["outcome"])
    ]
    return ran_tests, status_code


def calculate_test_time(test):
    return float(test.get('duration', 0) +
                 test['teardown'].get('duration', 0) +
                 test['setup'].get('duration', 0)) * 1000


def is_skipped(test_outcome):
    return test_outcome == "skipped" or test_outcome == "xfailed"


def is_passed(test_outcome):
    return test_outcome == "passed"


def _get_coverage_data():
    cov = CoverageData()
    cov.read()
    data = defaultdict(set)
    for file in cov.measured_files():
        pf = to_project(repo=Path("."), measured_file=Path(file))
        if not pf:
            continue
        for lineno, contexts in cov.contexts_by_lineno(file).items():
            for context in contexts:
                if context != "":
                    data[context[:context.rindex("|")]].add(Scope(
                        path=pf.as_posix(),
                        startline=lineno,
                        endline=lineno
                    ))

    return data


def to_project(repo: Path, measured_file: Path):
    with contextlib.suppress(ValueError):
        return measured_file.relative_to(repo.resolve())


def main(tests_to_skip, result_path, xml_path, args):
    args = ["-qq", "--cov", "--cov-context=test", "--json-report-file=none", "--color=yes", f"--junitxml={xml_path}",
            "-W", "ignore:Module already imported:pytest.PytestWarning"] + args
    tests, exit_code = run_tests(tests_to_skip, args)
    # dump to json with pydantic
    tests_dict = {}
    with open(result_path, "w") as f:
        for test in tests:
            tests_dict[test.name] = test.dict()
        json.dump(tests_dict, f)

    return exit_code


if __name__ == "__main__":
    _ = sys.argv[0]  # plugin.py
    result_path = sys.argv[1]  # "/tmp/pytest-result.json"
    xml_path = sys.argv[2] # "/tmp/pytest-junit.xml"
    tests_to_skip = json.loads(sys.argv[3]) # {TEST_NAME: PASSED}
    args = sys.argv[4:]     # "-v", "--cov", "--cov-context=test", "--json-report-file=none"
    sys.exit(main(tests_to_skip, result_path, xml_path, args))
