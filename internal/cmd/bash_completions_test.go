package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/direnv/direnv/v2/gzenv"
)

func TestBashCompletions(t *testing.T) {
	name := "./make '$(exit 9)`exit 8`\n.sh"
	args := []string{"/absolute path/make.sh", "autocompletions", "' $() ` ;\n"}
	value := gzenv.Marshal([]bashCompletion{{Name: name, Args: args}})
	registrations, err := loadBashCompletions(value)
	if err != nil || len(registrations) != 1 || registrations[0].Name != name {
		t.Fatalf("round trip: %v, %v", registrations, err)
	}
	apply, err := bashCompletionExport(value)
	if err != nil {
		t.Fatal(err)
	}
	clear, err := bashCompletionExport("")
	if err != nil {
		t.Fatal(err)
	}
	// Compare builtin output before and after restoring several kinds of specs.
	for _, definition := range []string{
		"complete -W " + BashEscape("one 'two' $(exit 7)") + " -o nospace",
		"complete -F existing -o default",
		"complete -C " + BashEscape("printf '%s\\n' original"),
	} {
		script := "set -eu\nexisting() { :; }\n" + definition + " -- " + BashEscape(name) + "\n" +
			"before=$(complete -p -- " + BashEscape(name) + ")\n" + apply +
			"complete -p -- " + BashEscape(name) + "\n" + clear +
			"[[ $(complete -p -- " + BashEscape(name) + ") == \"$before\" ]]\n" +
			clear + "[[ $(complete -p -- " + BashEscape(name) + ") == \"$before\" ]]\n"
		if output, err := exec.Command("bash", "-c", script).CombinedOutput(); err != nil {
			t.Fatalf("restoration: %v\n%s", err, output)
		}
	}
}

func TestInvalidBashCompletions(t *testing.T) {
	for _, value := range []string{
		"invalid", gzenv.Marshal("wrong type"),
		gzenv.Marshal([]bashCompletion{{Name: "", Args: []string{"/bin/test"}}}),
		gzenv.Marshal([]bashCompletion{{Name: "test"}}),
		gzenv.Marshal([]bashCompletion{{Name: "test", Args: []string{"relative"}}}),
		gzenv.Marshal([]bashCompletion{{Name: "test", Args: []string{"/bin/test", "\x00"}}}),
	} {
		if _, err := bashCompletionExport(value); err == nil {
			t.Errorf("accepted invalid declaration %q", value)
		}
	}
}

func TestBashCompletionCommand(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(t.TempDir(), "make space' dollar$ backtick`\n.sh")
	if err := os.WriteFile(executable, []byte("#!"+bash+"\nprintf '%s\\n' \"$1\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	args := []string{executable, "space quote' dollar$ backtick` newline\nend"}
	code, err := bashCompletionExport(gzenv.Marshal([]bashCompletion{{Name: "./make.sh", Args: args}}))
	if err != nil {
		t.Fatal(err)
	}
	// Read the installed -C argument and run it through Bash's completion API.
	script := "set -eu\n" + code + `
eval "set -- $(builtin complete -p ./make.sh)"
while (( $# )) && [[ $1 != -C ]]; do shift; done
[[ $# -ge 2 ]]
compgen -C "$2" -- ''
`
	output, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil || !strings.Contains(string(output), args[1]) {
		t.Fatalf("completion invocation: %v\n%s", err, output)
	}
}

func TestBashCompletionsOtherShells(t *testing.T) {
	// Other shells carry the declaration as data and retain their own completions.
	for _, shell := range []string{"zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s is not installed", shell)
			}
			marker := filepath.Join(t.TempDir(), "provider-ran")
			provider := filepath.Join(t.TempDir(), "provider")
			bash, err := exec.LookPath("bash")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(provider, []byte("#!"+bash+"\nprintf ran > "+BashEscape(marker)+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			value := gzenv.Marshal([]bashCompletion{{Name: "mytool", Args: []string{provider}}})
			before := Env{}
			after := Env{DIRENV_BASH_COMPLETIONS: value}
			enter, err := before.Diff(after).ToShell(DetectShell(shell))
			if err != nil {
				t.Fatal(err)
			}
			leave, err := after.Diff(before).ToShell(DetectShell(shell))
			if err != nil {
				t.Fatal(err)
			}
			var script string
			args := []string{"--no-config", "-c"}
			if shell == "zsh" {
				args = []string{"-f", "-c"}
				script = "set -eu\nautoload -Uz compinit; compinit -D -i\ncompdef _files mytool\n" + enter +
					"\n[[ -n $DIRENV_BASH_COMPLETIONS && $_comps[mytool] == _files ]]\n" + leave +
					"\n[[ -z ${DIRENV_BASH_COMPLETIONS-} && $_comps[mytool] == _files ]]\n"
			} else {
				script = "complete -c mytool -a original\nset before (complete)\n" + enter +
					"\nset -q DIRENV_BASH_COMPLETIONS; or exit 1\nset after (complete)\ntest \"$before\" = \"$after\"; or exit 1\n" + leave +
					"\nset -q DIRENV_BASH_COMPLETIONS; and exit 1\nset after (complete)\ntest \"$before\" = \"$after\"; or exit 1\n"
			}
			if output, err := exec.Command(path, append(args, script)...).CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("provider ran or marker check failed: %v", err)
			}
		})
	}
}
