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
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/pkg/v3/console"
)

var adminUserEnableCmd = cli.Command{
	Name:         "enable",
	Usage:        "enable user",
	Action:       mainAdminUserEnable,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET USERNAME

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable a disabled user 'foobar' on libreFS server.
     {{.Prompt}} {{.HelpName}} myserver foobar
`,
}

// checkAdminUserEnableSyntax - validate all the passed arguments
func checkAdminUserEnableSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainAdminUserEnable is the handle for "lc admin user enable" command.
func mainAdminUserEnable(ctx *cli.Context) error {
	checkAdminUserEnableSyntax(ctx)

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new libreFS Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.SetUserStatus(globalContext, args.Get(1), madmin.AccountEnabled)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to enable user")

	printMsg(userMessage{
		op:        ctx.Command.Name,
		AccessKey: args.Get(1),
	})

	return nil
}
