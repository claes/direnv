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
