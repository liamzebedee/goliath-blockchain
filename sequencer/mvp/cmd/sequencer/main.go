package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/liamzebedee/goliath/mvp/sequencer/cmd/commands"
)

func main() {
  subcommands.Register(subcommands.HelpCommand(), "")
  subcommands.Register(subcommands.FlagsCommand(), "")
  subcommands.Register(subcommands.CommandsCommand(), "")
  subcommands.Register(&commands.StartCmd{}, "")
  subcommands.Register(&commands.InitCmd{}, "")

  flag.Parse()
  ctx := context.Background()
  os.Exit(int(subcommands.Execute(ctx)))
}