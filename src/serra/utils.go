package serra

import "fmt"

var (
	Icon        = "\U0001F47C\U0001F3FB" // baby angel serra
	Reset       = "\033[0m"
	Background  = "\033[38;5;59m"
	CurrentLine = "\033[38;5;60m"
	Foreground  = "\033[38;5;231m"
	Comment     = "\033[38;5;103m"
	Cyan        = "\033[38;5;159m"
	Green       = "\033[38;5;120m"
	Orange      = "\033[38;5;222m"
	Pink        = "\033[38;5;212m"
	Purple      = "\033[38;5;183m"
	Red         = "\033[38;5;210m"
	Yellow      = "\033[38;5;229m"
)

func Banner() {
	LogMessage(fmt.Sprintf("%s Serra %v\n", Icon, version), "green")
}

// Colored output on commandline
func LogMessage(message string, color string) {
	if color == "red" {
		fmt.Printf("%s%s%s\n", Red, message, Reset)
	} else if color == "green" {
		fmt.Printf("%s%s%s\n", Green, message, Reset)
	} else if color == "purple" {
		fmt.Printf("%s%s%s\n", Purple, message, Reset)
	} else {
		fmt.Printf("%s\n", message)
	}
}
