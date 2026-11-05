// Command secretscan, git'teki (veya commit'e hazırlanan) dosyalarda gizli
// değer arar (SAFE-010). Bulgu varsa çıkış kodu 1'dir; değerler yazılmaz.
//
//	go run ./cmd/secretscan            # izlenen tüm dosyalar (CI)
//	go run ./cmd/secretscan -staged    # yalnızca commit'e hazırlananlar (pre-commit)
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/burakkuru5534/bunker/backend/internal/secretscan"
)

func main() {
	os.Exit(run(os.Args[1:], gitFiles, os.Stdout, os.Stderr))
}

// lister, taranacak dosyaları ve içeriklerini verir.
type lister func(staged bool) (map[string][]byte, error)

func run(args []string, list lister, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("secretscan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	staged := fs.Bool("staged", false, "yalnızca commit'e hazırlanan içerik (git index)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	files, err := list(*staged)
	if err != nil {
		fmt.Fprintf(stderr, "secretscan: %v\n", err)
		return 2
	}
	var findings []secretscan.Finding
	for path, content := range files {
		if secretscan.SkipPath(path) {
			continue
		}
		f, err := secretscan.Scan(path, bytes.NewReader(content))
		if err != nil {
			fmt.Fprintf(stderr, "secretscan: %s: %v\n", path, err)
			return 2
		}
		findings = append(findings, f...)
	}
	if len(findings) == 0 {
		fmt.Fprintf(stdout, "secretscan: %d dosya tarandı, gizli değer bulunmadı\n", len(files))
		return 0
	}
	for _, f := range findings {
		fmt.Fprintf(stdout, "%s:%d: olası gizli değer (%s)\n", f.Path, f.Line, f.Kind)
	}
	fmt.Fprintf(stdout, "secretscan: %d bulgu. Değeri dosyadan çıkarın (.env / Secret Manager); "+
		"sahte örnekse satıra %q ekleyin.\n", len(findings), secretscan.IgnoreMarker)
	return 1
}

// gitFiles, git'in izlediği dosyaları (staged: index'teki hâlleriyle) okur.
func gitFiles(staged bool) (map[string][]byte, error) {
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	rootDir := strings.TrimSpace(string(root))
	args := []string{"ls-files", "-z"}
	if staged {
		args = []string{"diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z"}
	}
	out, err := gitIn(rootDir, args...)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" {
			continue
		}
		var content []byte
		if staged {
			content, err = gitIn(rootDir, "show", ":"+p)
		} else {
			content, err = os.ReadFile(rootDir + "/" + p)
		}
		if err != nil {
			if os.IsNotExist(err) {
				continue // izlenen ama silinmiş dosya
			}
			return nil, fmt.Errorf("%s okunamadı: %w", p, err)
		}
		files[p] = content
	}
	return files, nil
}

func git(args ...string) ([]byte, error) { return gitIn("", args...) }

func gitIn(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %v %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
