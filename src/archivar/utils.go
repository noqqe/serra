package archivar

import "fmt"

const (
	challengeLimiter string = ">> "
	colorCyan        string = "\033[36m"
	colorGreen       string = "\033[32m"
	colorPurple      string = "\033[35m"
	colorRed         string = "\033[31m"
	colorReset       string = "\033[0m"
)

// Colored output on commandline
func LogMessage(message string, color string) {
	if color == "red" {
		fmt.Printf("%s%s%s\n", colorRed, message, colorReset)
	} else if color == "green" {
		fmt.Printf("%s%s%s\n", colorGreen, message, colorReset)
	} else {
		fmt.Printf("%s\n", message)
	}
}
