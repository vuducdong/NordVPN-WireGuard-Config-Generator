package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nordgen/internal/models"

	"golang.org/x/term"
)

const (
	maskChar          = '*'
	maxTokenReadBytes = 1024
)

type ConsoleManager struct {
	reader        *bufio.Reader
	output        *os.File
	isTerminal    bool
	spinnerActive atomic.Bool
	spinnerStop   chan struct{}
	spinnerExited chan struct{}
	progressMutex sync.Mutex
}

func NewConsoleManager() *ConsoleManager {
	return &ConsoleManager{
		reader:     bufio.NewReader(os.Stdin),
		output:     os.Stdout,
		isTerminal: term.IsTerminal(int(os.Stdout.Fd())),
	}
}

func (c *ConsoleManager) applyColor(code, text string) string {
	if !c.isTerminal {
		return text
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", code, text)
}

func (c *ConsoleManager) ClearScreen() {
	if !c.isTerminal {
		return
	}
	fmt.Fprint(c.output, "\033[2J\033[H")
}

func (c *ConsoleManager) Header() {
	line := strings.Repeat("=", 50)
	fmt.Fprintf(c.output, "\n%s\n", c.applyColor("37", line))
	fmt.Fprintf(c.output, "%s\n", c.applyColor("1;97", "  NordVPN Configuration Generator"))
	fmt.Fprintf(c.output, "%s\n\n", c.applyColor("37", line))
}

func (c *ConsoleManager) PromptSecret(message string) string {
	fmt.Fprintf(c.output, "%s: ", c.applyColor("1;97", message))

	if !c.isTerminal {
		input, err := c.reader.ReadString('\n')
		fmt.Fprintln(c.output)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(input)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		input, readErr := c.reader.ReadString('\n')
		fmt.Fprintln(c.output)
		if readErr != nil {
			return ""
		}
		return strings.TrimSpace(input)
	}

	var buf []byte
	oneByte := make([]byte, 1)
	totalRead := 0

	for totalRead < maxTokenReadBytes {
		n, readErr := os.Stdin.Read(oneByte)
		if n == 0 || readErr != nil {
			break
		}
		totalRead++
		b := oneByte[0]

		switch b {
		case '\r', '\n':
			fmt.Fprintln(c.output)
			term.Restore(int(os.Stdin.Fd()), oldState)
			return strings.TrimSpace(string(buf))
		case 3:
			fmt.Fprintln(c.output)
			term.Restore(int(os.Stdin.Fd()), oldState)
			return ""
		case 127, 8:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Fprint(c.output, "\b \b")
			}
		default:
			if b >= 32 {
				buf = append(buf, b)
				fmt.Fprintf(c.output, "%c", maskChar)
			}
		}
	}

	fmt.Fprintln(c.output)
	term.Restore(int(os.Stdin.Fd()), oldState)
	return strings.TrimSpace(string(buf))
}

func (c *ConsoleManager) PromptPreferences(defaults models.UserPreferences) models.UserPreferences {
	c.Info("Configuration Options (Enter for default)")

	dns := c.promptString("DNS IP", defaults.DNS)
	useIP := c.promptBool("Use IP for endpoints?", defaults.UseIP)
	keepaliveString := c.promptString("PersistentKeepalive", strconv.Itoa(defaults.Keepalive))

	keepalive, err := strconv.Atoi(keepaliveString)
	if err != nil {
		keepalive = defaults.Keepalive
	}

	excludeDedicated := c.promptBool("Exclude dedicated IP servers?", defaults.ExcludeDedicated)

	return models.UserPreferences{
		DNS:              dns,
		UseIP:            useIP,
		Keepalive:        keepalive,
		Groups:           defaults.Groups,
		ExcludeDedicated: excludeDedicated,
	}
}

func (c *ConsoleManager) promptString(message, defaultValue string) string {
	fmt.Fprintf(c.output, "%s [%s]: ", c.applyColor("1;97", message), c.applyColor("37", defaultValue))

	input, err := c.reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func (c *ConsoleManager) promptBool(message string, defaultValue bool) bool {
	defaultString := "y/N"
	if defaultValue {
		defaultString = "Y/n"
	}
	fmt.Fprintf(c.output, "%s [%s]: ", c.applyColor("1;97", message), c.applyColor("37", defaultString))

	input, err := c.reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultValue
	}
	return input == "y" || input == "yes"
}

func (c *ConsoleManager) StartStatus(message string) {
	if c.spinnerActive.Swap(true) {
		return
	}
	c.spinnerStop = make(chan struct{})
	c.spinnerExited = make(chan struct{})
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	go func() {
		defer close(c.spinnerExited)
		index := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-c.spinnerStop:
				fmt.Fprintf(c.output, "\r\033[K")
				return
			case <-ticker.C:
				fmt.Fprintf(c.output, "\r\033[K%s %s", c.applyColor("96", frames[index%len(frames)]), message)
				index++
			}
		}
	}()
}

func (c *ConsoleManager) StopStatus() {
	if !c.spinnerActive.Swap(false) {
		return
	}
	if c.spinnerStop != nil {
		close(c.spinnerStop)
		if c.spinnerExited != nil {
			<-c.spinnerExited
		}
		c.spinnerStop = nil
		c.spinnerExited = nil
	}
}

func (c *ConsoleManager) StartProgress(total int, message string) {
	if !c.isTerminal {
		return
	}
	c.progressMutex.Lock()
	defer c.progressMutex.Unlock()
	c.updateProgressInternal(0, total, message)
}

func (c *ConsoleManager) UpdateProgress(current, total int, message string) {
	if !c.isTerminal {
		return
	}
	c.progressMutex.Lock()
	defer c.progressMutex.Unlock()
	c.updateProgressInternal(current, total, message)
}

func (c *ConsoleManager) updateProgressInternal(current, total int, message string) {
	width := 30
	percent := float64(current) / float64(total)
	completed := int(float64(width) * percent)
	if completed > width {
		completed = width
	}

	bar := c.applyColor("92", strings.Repeat("█", completed)) + c.applyColor("37", strings.Repeat("░", width-completed))
	fmt.Fprintf(c.output, "\r\033[K%s %s [%s] %d/%d", c.applyColor("96", "→"), message, bar, current, total)
}

func (c *ConsoleManager) StopProgress() {
	if !c.isTerminal {
		return
	}
	fmt.Fprintf(c.output, "\n")
}

func (c *ConsoleManager) Success(message string) {
	fmt.Fprintf(c.output, "%s %s\n", c.applyColor("92", "✓"), message)
}

func (c *ConsoleManager) Fail(message string) {
	fmt.Fprintf(c.output, "%s %s\n", c.applyColor("91", "✗"), message)
}

func (c *ConsoleManager) Info(message string) {
	fmt.Fprintf(c.output, "%s %s\n", c.applyColor("96", "→"), message)
}

func (c *ConsoleManager) ShowKey(key string) {
	fmt.Fprintf(c.output, "\n%s\n", c.applyColor("1;97", "NordLynx Private Key"))
	fmt.Fprintf(c.output, "%s\n\n", c.applyColor("92", key))
}

func (c *ConsoleManager) Summary(outputPath string, stats models.GenerationStats, duration float64) {
	line := strings.Repeat("=", 40)
	fmt.Fprintf(c.output, "\n%s\n", c.applyColor("37", line))
	fmt.Fprintf(c.output, "%s\n", c.applyColor("1;97", "  Complete"))
	fmt.Fprintf(c.output, "%s\n", c.applyColor("37", line))
	fmt.Fprintf(c.output, "  Output Directory:  %s\n", c.applyColor("96", outputPath))
	fmt.Fprintf(c.output, "  Standard Configs:  %s\n", c.applyColor("96", strconv.Itoa(stats.Total)))
	fmt.Fprintf(c.output, "  Optimized Configs: %s\n", c.applyColor("96", strconv.Itoa(stats.Best)))
	fmt.Fprintf(c.output, "  Duration:          %s\n", c.applyColor("96", fmt.Sprintf("%.2fs", duration)))
	fmt.Fprintf(c.output, "%s\n\n", c.applyColor("37", line))
}

func (c *ConsoleManager) Wait() {
	fmt.Fprint(c.output, c.applyColor("37", "Press Enter to exit... "))
	c.reader.ReadBytes('\n')
}
