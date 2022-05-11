#!/bin/bash

declare -i exit_code
exit_code=0
unformatted_golang=$(gofmt -l .)
unformatted_shell=$(shfmt -f . | xargs shfmt -l)

if [ -n "$unformatted_golang" ]; then
	echo "ERROR: unformatted golang files:"
	echo "$unformatted_golang"
	exit_code=-1
fi

if [ -n "$unformatted_shell" ]; then
	echo "ERROR: unformatted script files:"
	echo "$unformatted_shell"
	exit_code=-1
fi

if [ $exit_code -ne 0 ]; then
	exit 1
fi

echo "SUCCEESS: test fmt"
