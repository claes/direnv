#!/usr/bin/env bash
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
export PATH="$root:$PATH"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export DIRENV_CONFIG="$work/config" XDG_DATA_HOME="$work/data"
unset DIRENV_DIR DIRENV_FILE DIRENV_DIFF DIRENV_WATCHES DIRENV_BASH_COMPLETIONS
mkdir -p "$work/a" "$work/b" "$DIRENV_CONFIG"
eval "$(direnv hook bash)"

update() { eval "$(direnv export bash)"; }
absent() { ! complete -p -- "$1" 2>/dev/null; }

cd "$work/a"
printf '%s\n' 'complete_add ./make.sh ignored' 'complete_add ./make.sh autocompletions' >.envrc
direnv allow
complete -W 'original words' -o nospace ./make.sh
original=$(complete -p ./make.sh)
_direnv_hook
first=$(complete -p ./make.sh)
[[ $first == *"$work/a/make.sh"* && $first == *autocompletions* ]]
# shellcheck disable=SC2154
[[ ${#_direnv_complete_names[@]} == 1 ]]
_direnv_hook
[[ $(complete -p ./make.sh) == "$first" ]]
sleep 1
direnv reload
_direnv_hook
[[ ${#_direnv_complete_names[@]} == 1 ]]

cd "$work/b"
printf '%s\n' 'complete_add ./make.sh other' 'complete_add ./second.sh args' >.envrc
direnv allow
_direnv_hook
switched=$(complete -p ./make.sh)
[[ $switched == *"$work/b/make.sh"* && $switched == *other* ]]
complete -p ./second.sh
cd "$work"
_direnv_hook
[[ $(complete -p ./make.sh) == "$original" ]]
absent ./second.sh
[[ -z ${DIRENV_BASH_COMPLETIONS-} ]]

# Reenter, change declarations, then fail a reload after declaring a completion.
cd "$work/a"
_direnv_hook
sleep 1
printf '%s\n' 'complete_add ./changed.sh args' >.envrc
direnv allow
_direnv_hook
[[ $(complete -p ./make.sh) == "$original" ]]
complete -p ./changed.sh
sleep 1
printf '%s\n' 'complete_add ./partial.sh args' 'exit 5' >.envrc
direnv allow
update
absent ./changed.sh
absent ./partial.sh
[[ $(complete -p ./make.sh) == "$original" ]]
[[ -z $(direnv export bash) ]]

# Blocked files must not install declarations.
direnv deny
update
absent ./partial.sh
cd "$work"
update

# Special characters remain literal at declaration and restoration time.
cd "$work/a"
sleep 1
name=$'./make space\'$(touch injected)`touch injected`\n.sh'
# shellcheck disable=SC2016
printf 'complete_add %q %q\n' "$name" 'arg $(touch injected)' >.envrc
direnv allow
_direnv_hook
complete -p -- "$name"
[[ ! -e injected ]]
cd "$work"
_direnv_hook
absent "$name"
[[ ! -e a/injected && ! -e injected ]]
echo 'Bash completion lifecycle tests passed'
