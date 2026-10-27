package forwardproxy

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/richargh/clamp/mitm-proxy/internal/kernel/environment"
)

type readyRecord struct {
	Schema  int    `json:"schema"`
	PID     int    `json:"pid"`
	Address string `json:"address"`
}

func writeReady(env *environment.Env, address net.Addr) error {
	record := readyRecord{Schema: 1, PID: env.PID, Address: address.String()}
	if err := json.NewEncoder(env.Stdout).Encode(record); err != nil {
		return fmt.Errorf("write readiness: %w", err)
	}
	return nil
}
