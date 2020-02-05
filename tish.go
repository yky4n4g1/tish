package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"sync"
)

type LineParseType int

const (
	Spase LineParseType = iota
	Arg
)

type Tish struct {
	Path     string
	PS1      string
	Username string
}

var StdinReader *bufio.Reader

func main() {
	fmt.Println("Welcom to tish.")
	StdinReader = bufio.NewReader(os.Stdin)
	user, err := user.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	shell := Tish{user.HomeDir, "> ", user.Name}
	mainLoop(&shell)
}

func mainLoop(shell *Tish) {
	outMutex := sync.Mutex{}
LOOP:
	for {
		fmt.Print(shell.PS1)
		wg := sync.WaitGroup{}
		rowLine, err := readCmd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tish: %s\n", err)
			return
		}
		name, args := lineParse(rowLine)
		switch name {
		case "exit":
			break LOOP
		case "":
			continue LOOP
		default:
			cmd := exec.Command(name, args...)
			stdoutPipe, _ := cmd.StdoutPipe()
			stderrPipe, _ := cmd.StderrPipe()
			wg.Add(2)
			go cmdOutput(bufio.NewScanner(stdoutPipe), os.Stdout, &outMutex, &wg)
			go cmdOutput(bufio.NewScanner(stderrPipe), os.Stderr, &outMutex, &wg)
			err := cmd.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "tish: %s\n", err)
			}
			wg.Wait()
		}
	}
}

func readCmd() (string, error) {
	buf := make([]byte, 0, 1000)
	for {
		line, p, err := StdinReader.ReadLine()
		if err != nil {
			return "", err
		}
		buf = append(buf, line...)
		if !p {
			break
		}
	}
	return string(buf), nil
}

func cmdOutput(scanner *bufio.Scanner, output *os.File, m *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	for scanner.Scan() {
		m.Lock()
		fmt.Fprintln(output, scanner.Text())
		m.Unlock()
	}
}

func lineParse(line string) (string, []string) {
	args := make([]string, 1, 20)
	argsIdx := 0
	stack := make([]LineParseType, 0, 5)
	for _, r := range line {
		if r == ' ' {
			if len(stack) > 0 && stack[len(stack)-1] != Spase {
				stack = append(stack[:len(stack)-1], Spase)
			}
		} else if r != '\n' {
			if len(stack) > 0 {
				if stack[len(stack)-1] == Spase {
					stack = append(stack[:len(stack)-1], Arg)
					args = append(args, "")
					argsIdx++
				}
			} else {
				stack = append(stack, Arg)
			}
			args[argsIdx] += string(r)
		}
	}
	return args[0], args[1:]
}
