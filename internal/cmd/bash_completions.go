package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/direnv/direnv/v2/gzenv"
)

type bashCompletion struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

func loadBashCompletions(value string) ([]bashCompletion, error) {
	var registrations []bashCompletion
	if value == "" {
		return registrations, nil
	}
	if err := gzenv.Unmarshal(value, &registrations); err != nil {
		return nil, fmt.Errorf("invalid Bash completions: %w", err)
	}
	for _, registration := range registrations {
		if registration.Name == "" || strings.ContainsRune(registration.Name, 0) || len(registration.Args) == 0 || !filepath.IsAbs(registration.Args[0]) {
			return nil, fmt.Errorf("invalid Bash completion registration")
		}
		for _, arg := range registration.Args {
			if strings.ContainsRune(arg, 0) {
				return nil, fmt.Errorf("invalid NUL in Bash completion argument")
			}
		}
	}
	return registrations, nil
}

// CmdBashCompleteCmd transports data only; no completion program runs during loading.
var CmdBashCompleteCmd = &Cmd{
	Name:    "bash-complete-cmd",
	Desc:    "Adds a Bash completion declaration",
	Args:    []string{"NAME", "EXECUTABLE", "ARGS..."},
	Private: true,
	Action: actionSimple(func(env Env, args []string) error {
		if len(args) < 3 {
			return fmt.Errorf("bash-complete-cmd requires a name and absolute executable path")
		}
		registrations, err := loadBashCompletions(env[DIRENV_BASH_COMPLETIONS])
		if err != nil {
			return err
		}
		registration := bashCompletion{Name: args[1], Args: args[2:]}
		updated := make([]bashCompletion, 0, len(registrations)+1)
		for _, existing := range registrations {
			if existing.Name != registration.Name {
				updated = append(updated, existing)
			}
		}
		updated = append(updated, registration)
		value := gzenv.Marshal(updated)
		if _, err := loadBashCompletions(value); err != nil {
			return err
		}
		fmt.Print(value)
		return nil
	}),
}

// Original definitions belong to this shell, so they must never be exported.
// Only complete -p output captured here is evaluated when restoring definitions.
const bashCompletionApply = `
_direnv_complete_apply() {
  local name saved command i;
  for i in "${!_direnv_complete_names[@]}"; do
    name=${_direnv_complete_names[i]};
    saved=${_direnv_complete_saved[i]};
    builtin complete -r -- "$name" 2>/dev/null || :;
    if [[ -n $saved ]]; then
      eval "builtin $saved";
    fi;
  done;
  _direnv_complete_names=();
  _direnv_complete_saved=();
  while (( $# )); do
    name=$1; command=$2; shift 2;
    saved=$(builtin complete -p -- "$name" 2>/dev/null) || saved=;
    if builtin complete -C "$command" -- "$name"; then
      _direnv_complete_names+=("$name");
      _direnv_complete_saved+=("$saved");
    fi;
  done;
};
_direnv_complete_apply`

func bashCompletionExport(value string) (string, error) {
	registrations, err := loadBashCompletions(value)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString(bashCompletionApply)
	for _, registration := range registrations {
		args := make([]string, len(registration.Args))
		for i, arg := range registration.Args {
			args[i] = BashEscape(arg)
		}
		out.WriteString(" " + BashEscape(registration.Name) + " " + BashEscape(strings.Join(args, " ")))
	}
	out.WriteString("; unset -f _direnv_complete_apply;\n")
	return out.String(), nil
}
