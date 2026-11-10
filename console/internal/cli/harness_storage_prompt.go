package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/richargh/clamp/console/internal/kernel/environment"
	"github.com/richargh/clamp/console/internal/session"
)

func promptHarnessStorage(env environment.Env, workspace string) (session.HarnessVolumeChoice, error) {
	_, _ = fmt.Fprintf(env.Stderr, "No Clamp auth preference found for:\n  %s\n\nUse:\n  1) global/shared credentials\n  2) per-project credentials\n\n", workspace)
	reader := bufio.NewReader(env.Stdin)
	for {
		_, _ = fmt.Fprint(env.Stderr, "Choice [1/2]: ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return session.NoChoiceMade, fmt.Errorf("unable to read harness storage preference: %w", err)
		}
		switch strings.TrimSpace(choice) {
		case "", "1":
			return session.SharedVolume, nil
		case "2":
			return session.ProjectVolume, nil
		default:
			_, _ = fmt.Fprintln(env.Stderr, "Please enter 1 or 2.")
		}
	}
}
