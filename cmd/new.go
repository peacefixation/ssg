package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var stdinReader = bufio.NewReader(os.Stdin)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new item or list",
	RunE:  runNewInteractive,
}

func init() {
	rootCmd.AddCommand(newCmd)
}

func runNewInteractive(cmd *cobra.Command, args []string) error {
	choices := []string{"item", "list"}
	idx, err := promptSelect("What would you like to create?", choices)
	if err != nil {
		return err
	}
	switch choices[idx] {
	case "item":
		return interactiveNewItem()
	case "list":
		return interactiveNewList()
	}
	return nil
}

func promptSelect(label string, options []string) (int, error) {
	fmt.Println(label)
	for i, opt := range options {
		fmt.Printf("%d) %s\n", i+1, opt)
	}
	for {
		fmt.Print("> ")
		line, err := stdinReader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("reading input: %w", err)
		}
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		fmt.Printf("Enter a number between 1 and %d\n", len(options))
	}
}

func promptInput(label, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return defaultVal, nil
	}
	return v, nil
}
