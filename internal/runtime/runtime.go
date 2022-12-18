package runtime

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Environment interface {
	Get() map[string]string
	WorkingDir() string
}

type EmptyEnv struct {
	baseDir string
}

func EnvWithDir(baseDir string) Environment {
	return &EmptyEnv{baseDir: baseDir}
}

func (e *EmptyEnv) Get() map[string]string {
	return make(map[string]string)
}

func (e *EmptyEnv) WorkingDir() string {
	return e.baseDir
}

func Exec(env Environment, command string, stdOut io.Writer) error {

	vars := env.Get()
	runCmd := ""
	for k, v := range vars {
		runCmd = runCmd + fmt.Sprintf("export %s=\"%s\" && ", k, v)
	}
	runCmd = runCmd + command

	fmt.Println(runCmd)

	cmd := exec.Command("/bin/sh", "-c", runCmd)
	fmt.Println(env.WorkingDir())
	if env.WorkingDir() != "" {
		cmd.Dir = env.WorkingDir()
	}
	cmd.Stdout = stdOut
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
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
		err := Exec(&EmptyEnv{}, "which openssl", &b)
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
