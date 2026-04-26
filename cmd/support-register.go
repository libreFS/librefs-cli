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
	"github.com/minio/cli"
)

var supportRegisterFlags = append([]cli.Flag{
	cli.StringFlag{
		Name:  "name",
		Usage: "Specify the name to associate to this libreFS cluster in SUBNET",
	},
}, subnetCommonFlags...)

var supportRegisterCmd = cli.Command{
	Name:               "register",
	Usage:              "register with libreFS subscription network",
	OnUsageError:       onUsageError,
	Action:             mainSupportRegister,
	Before:             setGlobalsFromContext,
	Flags:              supportRegisterFlags,
	CustomHelpTemplate: "Please use 'lc license register'",
}

func mainSupportRegister(_ *cli.Context) error {
	deprecatedError("lc license register")
	return nil
}
