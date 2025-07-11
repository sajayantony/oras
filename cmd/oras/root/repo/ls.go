/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/repository"
)

type repositoryOptions struct {
	option.Remote
	option.Common
	option.Format
	hostname  string
	namespace string
	last      string
}

func listCmd() *cobra.Command {
	var opts repositoryOptions
	cmd := &cobra.Command{
		Use:   "ls [flags] <registry>",
		Short: "List the repositories under the registry",
		Long: `List the repositories under the registry

Example - List the repositories under the registry:
  oras repo ls localhost:5000

Example - List the repositories under a namespace in the registry:
  oras repo ls localhost:5000/example-namespace

Example - List the repositories under the registry that include values lexically after last:
  oras repo ls --last "last_repo" localhost:5000

Example - [Experimental] List the repositories under the registry in JSON format:
  oras repo ls localhost:5000 --format json

Example - [Experimental] List the repositories under the registry using the given Go template:
  oras repo ls localhost:5000 --format go-template --template "{{.repositories}}"
`,
		Args:    oerrors.CheckArgs(argument.Exactly(1), "the target registry to list repositories from"),
		Aliases: []string{"list"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if opts.hostname, opts.namespace, err = repository.ParseRepoPath(args[0]); err != nil {
				return fmt.Errorf("could not parse repository path: %w", err)
			}
			return listRepository(cmd, &opts)
		},
	}

	cmd.Flags().StringVar(&opts.last, "last", "", "start after the repository specified by `last`")
	option.AddDeprecatedVerboseFlag(cmd.Flags())
	opts.SetTypes(option.FormatTypeText, option.FormatTypeJSON, option.FormatTypeGoTemplate)
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Remote)
}

func listRepository(cmd *cobra.Command, opts *repositoryOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	reg, err := opts.NewRegistry(opts.hostname, opts.Common, logger)
	if err != nil {
		return err
	}

	handler, err := display.NewRepoListHandler(opts.Printer, opts.Format, reg.Reference.Registry, opts.namespace)
	if err != nil {
		return err
	}
	err = reg.Repositories(ctx, opts.last, func(repos []string) error {
		for _, repo := range repos {
			if opts.namespace == "" || strings.HasPrefix(repo, opts.namespace) {
				// list repositories under the specified namespace
				if err := handler.OnRepositoryListed(repo); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		var repoErr error
		if opts.namespace != "" {
			repoErr = fmt.Errorf("could not list repositories for %q with prefix %q", reg.Reference.Host(), opts.namespace)
		} else {
			repoErr = fmt.Errorf("could not list repositories for %q", reg.Reference.Host())
		}
		return errors.Join(repoErr, err)
	}

	return handler.Render()
}
