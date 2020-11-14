.PHONY: test
test:
	# Set up variables.
	$(eval STD_OUT_LOG := ./zdevelop/tests/_reports/test_stdout.txt)
	$(eval STD_ERR_LOG := ./zdevelop/tests/_reports/test_stderr.txt)
	$(eval FULL_LOG := ./zdevelop/tests/_reports/test_full.txt)
	$(eval COVERAGE_LOG := ./zdevelop/tests/_reports/coverage.out)
	$(eval JUNIT_REPORT := ./zdevelop/tests/_reports/junit.xml)
	$(eval TEST_REPORT := ./zdevelop/tests/_reports/test_results.html)
	$(eval COVERAGE_REPORT := ./zdevelop/tests/_reports/coverage.html)
	# make the reports directory
	-mkdir ./zdevelop/tests/_reports
	# Clear the output files.
	echo > $(STD_OUT_LOG)
	echo > $(STD_ERR_LOG)
	# Run tests. I honestly don't quite understand the piping bullshit that has to
	# happen here to send stdout and stderr to tee separately ( in order to
	# both save and display them ), but the internet says this is the solution and it
	# works.
	-python3 ./zdevelop/make_scripts/go_make_test.py
	cat $(FULL_LOG) | go-junit-report > $(JUNIT_REPORT)
	# Open Reports
	-xunit-viewer -r $(JUNIT_REPORT) -o $(TEST_REPORT)
	-go tool cover -html=$(COVERAGE_LOG) -o $(COVERAGE_REPORT)
	-python3 ./zdevelop/make_scripts/py_open_test_reports.py

.PHONY: lint
lint:
	-revive -config revive.toml ./...

.PHONY: format
format:
	-gofmt -s -w ./
	-gofmt -s -w ./zdevelop/tests

.PHONY: venv
venv:
ifeq ($(py), )
	$(eval PY_PATH := python3)
else
	$(eval PY_PATH := $(py))
endif
	$(eval VENV_PATH := $(shell $(PY_PATH) ./zdevelop/make_scripts/go_make_venv.py))
	@echo "venv created! To enter virtual env, run:"
	@echo ". ~/.bash_profile"
	@echo "then run:"
	@echo "$(VENV_PATH)"

.PHONY: install-dev
install-dev:
	pip install --upgrade pip
	pip install --no-cache-dir -e .[build,doc,dev,lint,test]
	go mod tidy

# Installs command line tools for development
.PHONY: install-tools
install-globals:
	# Creates html report of tests.
	-go install github.com/ains/go-test-html
	# Creates API doc server.
	-go install golang.org/x/tools/cmd/godoc
	# Downloads module APIs from API server.
	-go install github.com/illuscio-dev/docmodule-go
	# Linter
	-go install github.com/mgechev/revive
	# Converts to junit for making html reports
	-go install github.com/jstemmer/go-junit-report
	# Converts junit reports into pretty html
	-npm i -g xunit-viewer

# Creates docs.
.PHONY: doc
doc:
	rm -rf ./zdocs/build
	mkdir ./zdocs/build
	# Rip API docs from godoc. This tools spins up a godoc server and downloads
	# module docs
	docmodule-go
	python setup.py build_sphinx -E
	sleep 1
	-python3 ./zdevelop/make_scripts/open_docs.py
	# Remove Deleted files from git
	git add -u
	# Add any new files to git
	git add zdocs/*

.PHONY: name
name:
	$(eval PATH_NEW := $(shell python3 ./zdevelop/make_scripts/go_make_name.py $(n)))
	@echo "library renamed! to switch your current directory, use the following \
	command:\ncd '$(PATH_NEW)'"

.PHONY: proto
proto:
	python3 ./zdevelop/make_scripts/go_gen_proto.py
