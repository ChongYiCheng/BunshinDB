package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
		cmdString, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		err = runCommand(cmdString)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
func runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
    if len(arrCommandStr)>=1{
    	switch arrCommandStr[0] {
    	case "exit":
    		os.Exit(0)
    		// add another case here for custom commands.
        case "help":
            fmt.Printf("Here are the list of commands\n help - Shows lists of commands\n exit - quits program\n")
        default:
    	cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
    	cmd.Stderr = os.Stderr
    	cmd.Stdout = os.Stdout
    	return cmd.Run()
    }
}
    return nil
}
