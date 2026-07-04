# VHS tape — regenerates demo.gif (https://github.com/charmbracelet/vhs)
# Run locally with `vhs demo.tape` or via the "demo" GitHub Actions workflow.
Output demo.gif

Set FontSize 15
Set Width 940
Set Height 560
Set Padding 12
Set TypingSpeed 70ms

Type "dockup"
Enter
Sleep 3s

Down@400ms 4
Sleep 1s

# Open the read-only security audit
Enter
Sleep 5s
Escape
Sleep 1s

# Show status & disk usage
Down
Enter
Sleep 4s
Escape
Sleep 1s

Type "q"
Sleep 1s
