package util

import (
	"bytes"
	"fmt"
	"math/rand"
	"os/exec"
	"runtime"
	"strings"
	"time"

	rt "github.com/OpenPaas/openpaas/internal/runtime"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))] //nolint
	}
	return string(b)
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func HasDependencies() bool {
	dependencies := []string{
		"consul",
		"nomad",
		"vault",
		"ansible-playbook",
		"cfssl",
		"openssl",
	}

	var b bytes.Buffer
	if runtime.GOOS == "darwin" {
		err := rt.Exec(&rt.EmptyEnv{}, "which openssl", &b)
		if err != nil {
			fmt.Println("openssl not present")
			fmt.Println(err)
			return false
		}
		if strings.Contains(b.String(), "/usr/bin/openssl") {
			fmt.Println("openssl is required, however on MacOS, the default MacOS is not compatible with our requirements. Please install openssl with brew or nix and ensure it is on the PATH")
			return false
		}
	}

	missing := []string{}
	for _, v := range dependencies {
		if !commandExists(v) {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		fmt.Println("Local dependencies unsatisfied.\n Please install the following applications with your package manager of choice and ensure they are on the PATH:")
	}
	for _, v := range missing {
		fmt.Printf("- %s\n", v)
	}
	return len(missing) == 0
}
