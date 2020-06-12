all: test format build

build:
	pip3 install --no-cache-dir .

test: test-pylint test-flake8 test-coverage

test-flake8:
	flake8 .

test-pylint:
	pylint */

test-coverage:
	coverage run -m dummy tests
	coverage report -m

format:
	echo "no formatting for $$PWD"
