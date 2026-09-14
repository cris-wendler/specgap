#!/bin/sh
# Checks the writing in tracked files: no marketing language, no filler,
# and no claim that a coding tool wrote something.
#
# This script names the phrases it looks for, so it is not checked
# against itself. The licence is an official text and is left alone.
set -u
cd "$(dirname "$0")/.." || exit 2

status=0
marketing='AI[- ]powered|agentic|revolutionary|next[- ]generation|game[- ]changing|seamless|supercharge|cutting[- ]edge|blazing|effortless|\bmagic\b|leverage|harness the power|best[- ]in[- ]class|enterprise[- ]grade|battle[- ]tested|robust|utilize|empower|streamline|elevate|unleash|paradigm|synergy|holistic|turnkey|world[- ]class|industry[- ]leading|unparalleled|10x'
filler='in today.s|look no further|at its core|it is important to note|the possibilities are endless|deep dive|\bdelve\b|this powerful|say goodbye to'
attribution='generated (by|with)|created with (claude|copilot|chatgpt|an ai)|written by an ai|co-authored-by:'
em=$(printf '\342\200\224')
en=$(printf '\342\200\223')

report() {
	printf '%s  %s\n' "$1" "$2"
	status=1
}

for f in $(git ls-files | grep -v -E '^(LICENSE|scripts/lint-copy\.sh)$'); do
	[ -f "$f" ] || continue
	for line in $(grep -inE "$marketing" "$f" 2>/dev/null | cut -d: -f1); do
		report "$f:$line" "marketing language"
	done
	for line in $(grep -inE "$filler" "$f" 2>/dev/null | cut -d: -f1); do
		report "$f:$line" "filler"
	done
	for line in $(grep -inE "$attribution" "$f" 2>/dev/null | cut -d: -f1); do
		report "$f:$line" "tool attribution"
	done
	for line in $(grep -nF "$em" "$f" 2>/dev/null | cut -d: -f1); do
		report "$f:$line" "em dash"
	done
	for line in $(grep -nF "$en" "$f" 2>/dev/null | cut -d: -f1); do
		report "$f:$line" "en dash"
	done
done

if [ "$status" -eq 0 ]; then
	echo "lint-copy: no problems found"
fi
exit "$status"
