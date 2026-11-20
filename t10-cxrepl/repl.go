package repl

//put the repo command stuff here

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/skycoin/skycoin-cx/t3-ast"         // thin: only InREPL; no depth gain
	"github.com/skycoin/skycoin-cx/t4-execute"     // thin: only RunCompiled; no depth gain
	cxinit "github.com/skycoin/skycoin-cx/t7-init" // thin: only MakeProgram, RegisterPackages; no depth gain

	"github.com/skycoin/skycoin-cx/t4-actions"                               // thin: only AST, LineNo; no depth gain
	cxpartialparsing "github.com/skycoin/skycoin-cx/t5-cxpartialparsing"     // thin: only Parse, Program; no depth gain
	cxparsingcompletor "github.com/skycoin/skycoin-cx/t8-cxparsingcompletor" // thin: only Lexer, NewLexer, Parse; no depth gain
	cxparsering "github.com/skycoin/skycoin-cx/t9-cxparsing"                 // thin: only AddInitFunction; could move AddInitFunction to t4-actions
)

const VERSION = "0.8.0"

// VIOLATION: state outside State. Owner: t10-cxrepl; moves to ReplMng in phase 2.
var ReplTargetFn string = ""

// VIOLATION: state outside State. Owner: t10-cxrepl; moves to ReplMng in phase 2.
var ReplTargetStrct string = ""

// VIOLATION: state outside State. Owner: t10-cxrepl; moves to ReplMng in phase 2.
var ReplTargetMod string = ""

func unsafeEval(code string) (out string) {
	var lexer *cxparsingcompletor.Lexer
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("%v", r)
			// lexer.Stop()
			return
		}
	}()

	// storing strings sent to standard output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	actions.LineNo = 0

	// VIOLATION: init chaining. Init 7-8 again, a fresh program on REPL reset.
	actions.AST = cxinit.MakeProgram()
	cxinit.RegisterPackages(actions.AST)
	cxpartialparsing.Program = actions.AST

	cxpartialparsing.Parse(code)

	actions.AST = cxpartialparsing.Program

	lexer = cxparsingcompletor.NewLexer(bytes.NewBufferString(code))
	cxparsingcompletor.Parse(lexer)
	//yyParse(lexer)
	err := cxparsering.AddInitFunction(actions.AST)
	if err != nil {
		return fmt.Sprintf("%s", err)
	}

	//err = actions.AST.RunCompiled(0, nil);
	err = execute.RunCompiled(actions.AST, 0, nil)
	if err != nil {
		// VIOLATION: init chaining. Init 7-8 again, a fresh program on REPL reset.
		actions.AST = cxinit.MakeProgram()
		cxinit.RegisterPackages(actions.AST)
		return fmt.Sprintf("%s", err)
	}
	//Tod: If error equals nill?

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stdout = old // restoring the real stdout
	out = <-outC

	// VIOLATION: init chaining. Init 7-8 again, a fresh program on REPL reset.
	actions.AST = cxinit.MakeProgram()
	cxinit.RegisterPackages(actions.AST)
	return out
}

func Eval(code string) string {
	runtime.GOMAXPROCS(2)
	ch := make(chan string, 1)

	var result string

	go func() {
		result = unsafeEval(code)
		ch <- result
	}()

	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()

	select {
	case <-ch:
		return result
	case <-timer.C:
		// VIOLATION: init chaining. Init 7-8 again, a fresh program on REPL reset.
		actions.AST = cxinit.MakeProgram()
		cxinit.RegisterPackages(actions.AST)
		return "Timed out."
	}
}

func Repl() {
	fmt.Println("CX", VERSION)
	fmt.Println("More information about CX is available at http://cx.skycoin.com/ and https://github.com/skycoin/cx/")

	ast.InREPL = true

	// fi := bufio.NewReader(os.NewFile(0, "stdin"))
	fi := bufio.NewReader(os.Stdin)
	// scanner := bufio.NewScanner(os.Stdin)

	for {
		var inp string
		var ok bool

		printPrompt()

		if inp, ok = readline(fi); ok {
			if ReplTargetFn != "" {
				inp = fmt.Sprintf(":func %s {\n%s\n}\n", ReplTargetFn, inp)
			}
			if ReplTargetMod != "" {
				inp = fmt.Sprintf("%s", inp)
			}
			if ReplTargetStrct != "" {
				inp = fmt.Sprintf(":struct %s {\n%s\n}\n", ReplTargetStrct, inp)
			}

			b := bytes.NewBufferString(inp)

			cxparsingcompletor.Parse(cxparsingcompletor.NewLexer(b))
			//yyParse(NewLexer(b))
		} else {
			if ReplTargetFn != "" {
				ReplTargetFn = ""
				continue
			}

			if ReplTargetStrct != "" {
				ReplTargetStrct = ""
				continue
			}

			if ReplTargetMod != "" {
				ReplTargetMod = ""
				continue
			}

			fmt.Printf("\nBye!\n")
			break
		}
	}
}

func printPrompt() {
	if ReplTargetMod != "" {
		fmt.Println(fmt.Sprintf(":package %s ...", ReplTargetMod))
		fmt.Printf("* ")
	} else if ReplTargetFn != "" {
		fmt.Println(fmt.Sprintf(":func %s {...", ReplTargetFn))
		fmt.Printf("\t* ")
	} else if ReplTargetStrct != "" {
		fmt.Println(fmt.Sprintf(":struct %s {...", ReplTargetStrct))
		fmt.Printf("\t* ")
	} else {
		fmt.Printf("* ")
	}
}

// ----------------------------------------------------------------
//                     Utility functions

func readline(fi *bufio.Reader) (string, bool) {
	s, err := fi.ReadString('\n')

	s = strings.Replace(s, "\n", "", -1)
	s = strings.Replace(s, "\r", "", -1)

	for _, ch := range s {
		if ch == rune(4) {
			err = io.EOF
			break
		}
	}

	if err != nil {
		return "", false
	}

	return s, true
}
