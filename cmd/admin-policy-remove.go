// Copyright (c) 2015-2022 libreFS, Inc.
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
	"github.com/fatih/color"
	"github.com/libreFS/librefs-cli/pkg/probe"
	"github.com/libreFS/pkg/v3/console"
	"github.com/minio/cli"
)

var adminPolicyRemoveCmd = cli.Command{
	Name:         "remove",
	ShortName:    "rm",
	Usage:        "remove an IAM policy",
	Action:       mainAdminPolicyRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME

POLICYNAME:
  Name of the canned policy on libreFS server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove 'writeonly' policy on libreFS server.
     {{.Prompt}} {{.HelpName}} myserver writeonly
`,
}

// checkAdminPolicyRemoveSyntax - validate all the passed arguments
func checkAdminPolicyRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainAdminPolicyRemove is the handle for "lc admin policy remove" command.
func mainAdminPolicyRemove(ctx *cli.Context) error {
	checkAdminPolicyRemoveSyntax(ctx)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new libreFS Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	fatalIf(probe.NewError(client.RemoveCannedPolicy(globalContext, args.Get(1))).Trace(args...), "Unable to remove policy")

	printMsg(userPolicyMessage{
		op:     ctx.Command.Name,
		Policy: args.Get(1),
	})

	return nil
}
