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
printf '%s\n' 'bash_complete_cmd ./make.sh ./make.sh ignored' 'bash_complete_cmd ./make.sh ./make.sh autocompletions' >.envrc
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
printf '%s\n' 'bash_complete_cmd ./make.sh ./make.sh other' 'bash_complete_cmd ./second.sh ./second.sh args' >.envrc
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
printf '%s\n' 'bash_complete_cmd ./changed.sh ./changed.sh args' >.envrc
direnv allow
_direnv_hook
[[ $(complete -p ./make.sh) == "$original" ]]
complete -p ./changed.sh
sleep 1
printf '%s\n' 'bash_complete_cmd ./partial.sh ./partial.sh args' 'exit 5' >.envrc
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
printf 'bash_complete_cmd %q %q %q\n' "$name" "$name" 'arg $(touch injected)' >.envrc
direnv allow
_direnv_hook
complete -p -- "$name"
[[ ! -e injected ]]
cd "$work"
_direnv_hook
absent "$name"
[[ ! -e a/injected && ! -e injected ]]

# Lookup names are independent of provider paths. A provider can be shared.
cd "$work/a"
sleep 1
provider='./completion provider'
export COMPLETION_TEST_MARKER="$work/provider-ran"
printf '#!%s\n' "$(command -v bash)" >"$provider"
cat >>"$provider" <<'EOF'
: > "$COMPLETION_TEST_MARKER"
printf '%s\n' "$@"
EOF
chmod +x "$provider"
argument=$'literal space\' $() ` ;\nargument'
printf 'bash_complete_cmd mytool %q %q\nbash_complete_cmd mt %q\n' \
  "$provider" "$argument" "$provider" >.envrc
direnv allow
_direnv_hook
[[ ! -e $COMPLETION_TEST_MARKER ]]
[[ $(complete -p mytool) == *"$work/a/completion provider"* ]]
[[ $(complete -p mt) == *"$work/a/completion provider"* ]]
absent "$provider"
# Only builtin-generated output is evaluated to read back the -C argument.
eval "set -- $(complete -p mytool)"
while (($#)) && [[ $1 != -C ]]; do shift; done
[[ $# -ge 2 ]]
actual=$(compgen -C "$2" -- prefix 2>/dev/null)
expected=$(printf '%s\n' "$argument" compgen prefix '')
[[ $actual == "$expected" && -e $COMPLETION_TEST_MARKER ]]
cd "$work"
_direnv_hook
absent mytool
absent mt

# Invalid calls fail without changing the declaration data.
(
  # shellcheck disable=SC1091
  source "$root/stdlib.sh"
  before=${DIRENV_BASH_COMPLETIONS-}
  bash_complete_cmd && exit 1
  bash_complete_cmd mytool && exit 1
  bash_complete_cmd '' ./provider && exit 1
  bash_complete_cmd mytool '' && exit 1
  [[ ${DIRENV_BASH_COMPLETIONS-} == "$before" ]]
  if declare -F complete_add; then exit 1; fi
)
echo 'Bash completion lifecycle tests passed'
