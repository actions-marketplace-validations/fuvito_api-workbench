Output docs/demo.gif

Set Shell "bash"
Set FontSize 13
Set Width 900
Set Height 450
Set Theme "Dracula"
Set Framerate 24
Set PlaybackSpeed 1.0

# Navigate to the repo root so relative paths work in both CI and local runs
Type "cd $(git rev-parse --show-toplevel)"
Enter
Sleep 300ms
Type "clear"
Enter
Sleep 300ms

# --- Part 1: multi-step scenario run ---
Type "clrsig run examples/scenarios/user-lifecycle.yaml --env examples/environments/local.yaml"
Sleep 600ms
Enter
Sleep 5s

# --- Part 2: spec conformance check ---
Type "clrsig conform examples/server/openapi.yaml --base-url http://localhost:8080 --coverage --env examples/environments/demo-conform.yaml"
Sleep 600ms
Enter
Sleep 6s

# Hold the final frame
Sleep 3s
