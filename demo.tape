# Recorded with fictional demo data only (SUNDIAL_SOURCE=fake).
Output docs/demo.gif
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Theme "Catppuccin Macchiato"
Set TypingSpeed 150ms
Env SUNDIAL_SOURCE "fake"

# Agenda: move down two events, open and close the detail pane
Type "./bin/sundial" Enter
Sleep 3s
Type "j" Sleep 1s
Type "j" Sleep 1.5s
Enter Sleep 3.5s
Escape Sleep 1.5s

# Week: switch view and step through the day's events
Type "w" Sleep 3s
Type "j" Sleep 1s
Type "j" Sleep 1s
Type "j" Sleep 2s

# Month
Type "m" Sleep 3s

# Command palette: jump to next Friday
Ctrl+K Sleep 1.5s
Type "next fri" Sleep 2.5s
Enter Sleep 3s
Type "q"
