#!/bin/bash

# This scripts counts the lines of code (LOC) and comments in Go source files
# within this project directory. It uses the commandline tool "cloc".
# Requires `cloc` to be installed (e.g., `sudo apt install cloc` or `brew install cloc`).
# Modified from: https://schneegans.github.io/tutorials/2022/04/18/badges
#
# Usage:
#   Run from the repository root:
#     ./cloc.sh
#
# Default Output:
#   Displays a summary of code statistics:
#     Total lines of code:                   <value>k
#     Lines of source code:                  <value>k
#     Lines of comments (source code):       <value>k
#     Lines of test code:                    <value>k
#     Comment Percentage:                    <value>%
#     Test Percentage:                       <value>%
#
# Flags for Specific Metrics:
#   You can request individual metrics using the following flags:
#     --loc             : Lines of source code (Go files, excluding tests).
#     --comments        : Lines of comments in source code.
#     --percentage      : Comment percentage in source code.
#     --test-loc        : Lines of test code (_test.go files + tests/integration/ dir).
#     --test-percentage : Percentage of test code compared to total code.
#     --total-loc       : Total lines of code (source + test).
#
# Example:
#   ./cloc.sh --test-percentage
#   # Output: 19.0 (example value)

# Get the location of this script.
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

# Run cloc for source code - this counts code lines, blank lines and comment lines
# for the specified languages, excluding test files.
# We are only interested in the summary, therefore the tail -1
SUMMARY_SRC="$(cloc "${SCRIPT_DIR}" --include-lang="Go" --not-match-f="_test\.go$" --not-match-d="tests/integration" --md | tail -1)"

# Run cloc for test files ending in _test.go
SUMMARY_TEST_FILES="$(cloc "${SCRIPT_DIR}" --include-lang="Go" --match-f='_test\.go$' --md | tail -1)"

# Run cloc for the tests/integration directory if it exists
SUMMARY_TEST_DIR=""
if [[ -d "${SCRIPT_DIR}/tests/integration" ]]; then
  SUMMARY_TEST_DIR="$(cloc "${SCRIPT_DIR}/tests/integration" --include-lang="Go" --md | tail -1)"
fi


# The SUMMARY strings are lines of a markdown table and look like this:
# SUM:|files|blank|comment|code
# We use the following command to split it into an array.
IFS='|' read -r -a TOKENS_SRC <<< "$SUMMARY_SRC"
IFS='|' read -r -a TOKENS_TEST_FILES <<< "$SUMMARY_TEST_FILES"
IFS='|' read -r -a TOKENS_TEST_DIR <<< "$SUMMARY_TEST_DIR"

# Store the individual tokens for better readability.
# Source Code
NUMBER_OF_FILES_SRC=${TOKENS_SRC[1]:-0} # Default to 0 if empty
COMMENT_LINES_SRC=${TOKENS_SRC[3]:-0}
LINES_OF_CODE_SRC=${TOKENS_SRC[4]:-0}

# Test Code (_test.go files)
LINES_OF_CODE_TEST_FILES=${TOKENS_TEST_FILES[4]:-0}

# Test Code (tests/integration dir)
LINES_OF_CODE_TEST_DIR=${TOKENS_TEST_DIR[4]:-0}

# Total Test Code
LINES_OF_TEST_CODE=$((LINES_OF_CODE_TEST_FILES + LINES_OF_CODE_TEST_DIR))

# Total Code (Source + Test)
TOTAL_LINES_OF_CODE=$((LINES_OF_CODE_SRC + LINES_OF_TEST_CODE))


# Print all results if no arguments are given.
if [[ $# -eq 0 ]] ; then
  awk -v loc_src=$LINES_OF_CODE_SRC \
      -v comments_src=$COMMENT_LINES_SRC \
      -v loc_test=$LINES_OF_TEST_CODE \
      -v loc_total=$TOTAL_LINES_OF_CODE \
      'BEGIN {
          label_width = 35 # Define a width for the labels
          printf "%-*s %6.1fk\n", label_width, "Total lines of code:", loc_total/1000;
          printf "%-*s %6.1fk\n", label_width, "Lines of source code:", loc_src/1000;
          printf "%-*s %6.1fk\n", label_width, "Lines of comments (source code):", comments_src/1000;
          printf "%-*s %6.1fk\n", label_width, "Lines of test code:", loc_test/1000;
          if (loc_src + comments_src > 0) {
            printf "%-*s %6.1f%%\n", label_width, "Comment Percentage:", 100*comments_src/(loc_src + comments_src);
          } else {
            printf "%-*s %6s\n", label_width, "Comment Percentage:", "N/A"; # Adjusted N/A alignment
          }
          if (loc_src + loc_test > 0) { 
            printf "%-*s %6.1f%%\n", label_width, "Test Percentage:", 100*loc_test/(loc_src + loc_test);
          } else {
            printf "%-*s %6s\n", label_width, "Test Percentage:", "N/A"; # Adjusted N/A alignment
          }
      }'
  exit 0
fi

# --- Argument Parsing ---

# Show lines of source code if --loc is given.
if [[ $* == *--loc* ]]
then
  awk -v a=$LINES_OF_CODE_SRC \
      'BEGIN {printf "%.1fk\n", a/1000}'
fi

# Show lines of comments if --comments is given.
if [[ $* == *--comments* ]]
then
  awk -v a=$COMMENT_LINES_SRC \
      'BEGIN {printf "%.1fk\n", a/1000}'
fi

# Show percentage of comments if --percentage is given.
if [[ $* == *--percentage* ]]
then
  awk -v a=$COMMENT_LINES_SRC -v b=$LINES_OF_CODE_SRC \
      'BEGIN {if (a+b > 0) printf "%.1f\n", 100*a/(a+b); else print "N/A"}'
fi

# Show lines of test code if --test-loc is given.
if [[ $* == *--test-loc* ]]
then
  awk -v a=$LINES_OF_TEST_CODE \
      'BEGIN {printf "%.1fk\n", a/1000}'
fi

# Show test percentage if --test-percentage is given.
if [[ $* == *--test-percentage* ]]
then
  awk -v a=$LINES_OF_TEST_CODE -v b=$LINES_OF_CODE_SRC \
      'BEGIN {if (a+b > 0) printf "%.1f\n", 100*a/(a+b); else print "N/A"}'
fi

# Show total lines of code if --total-loc is given.
if [[ $* == *--total-loc* ]]
then
  awk -v a=$TOTAL_LINES_OF_CODE \
      'BEGIN {printf "%.1fk\n", a/1000}'
fi