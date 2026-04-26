// Copyright (c) 2022 libreFS, Inc.
//
// This file is part of libreFS Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/libreFS/librefs-cli/pkg/probe"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/pkg/v3/console"
)

var adminRebalanceStartCmd = cli.Command{
	Name:         "start",
	Usage:        "start rebalance operation",
	Action:       mainAdminRebalanceStart,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
xEXAMPLES:
  1. Start rebalance on a libreFS deployment with alias myserver
     {{.Prompt}} {{.HelpName}} myserver
`,
}

type rebalanceStartMsg struct {
	Status string `json:"status"`
	Target string `json:"url"`
	ID     string `json:"id"`
}

func (r rebalanceStartMsg) JSON() string {
	r.Status = "success"
	b, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal to JSON")
	return string(b)
}

func (r rebalanceStartMsg) String() string {
	return console.Colorize("rebalanceStartMsg", fmt.Sprintf("Rebalance started for %s", r.Target))
}

func mainAdminRebalanceStart(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1)
	}

	console.SetColor("rebalanceStartMsg", color.New(color.FgGreen))

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new libreFS Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client")

	id, e := client.RebalanceStart(globalContext)
	fatalIf(probe.NewError(e), "Unable to start rebalance")

	printMsg(rebalanceStartMsg{
		Target: aliasedURL,
		ID:     id,
	})

	return nil
}
