package command

import (
	"fmt"
	"time"

	"github.com/dnaeon/gru/catalog"
	"github.com/dnaeon/gru/task"

	"github.com/codegangsta/cli"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uitable"
)

// NewRunCommand creates a new sub-command for submitting
// tasks to minions
func NewRunCommand() cli.Command {
	cmd := cli.Command{
		Name:   "run",
		Usage:  "send task to minion(s)",
		Action: execRunCommand,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "is-concurrent",
				Usage: "flag task as concurrent",
			},
			cli.StringFlag{
				Name:  "with-classifier",
				Value: "",
				Usage: "match minions with given classifier pattern",
			},
		},
	}

	return cmd
}

// Executes the "run" command
func execRunCommand(c *cli.Context) {
	if len(c.Args()) < 1 {
		displayError(errNoModuleName, 64)
	}

	// Create the task that we send to our minions
	main := c.Args()[0]
	katalog, err := catalog.Load(main, c.GlobalString("modulepath"))
	if err != nil {
		displayError(err, 1)
	}

	t := task.New(katalog)
	t.IsConcurrent = c.Bool("is-concurrent")

	client := newEtcdMinionClientFromFlags(c)

	cFlag := c.String("with-classifier")
	minions, err := parseClassifierPattern(client, cFlag)

	if err != nil {
		displayError(err, 1)
	}

	numMinions := len(minions)
	if numMinions == 0 {
		displayError(errNoMinionFound, 1)
	}

	fmt.Printf("Found %d minion(s) for task processing\n\n", numMinions)

	// Progress bar to display while submitting task
	progress := uiprogress.New()
	bar := progress.AddBar(numMinions)
	bar.AppendCompleted()
	bar.PrependElapsed()
	progress.Start()

	// Number of minions to which submitting the task has failed
	failed := 0

	// Submit task to minions
	fmt.Println("Submitting task to minion(s) ...")
	for _, m := range minions {
		err = client.MinionSubmitTask(m, t)
		if err != nil {
			fmt.Printf("Failed to submit task to %s: %s\n", m, err)
			failed += 1
		}
		bar.Incr()
	}

	// Stop progress bar and sleep for a bit to make sure the
	// progress bar gets updated if we were too fast for it
	progress.Stop()
	time.Sleep(time.Millisecond * 100)

	// Display task report
	fmt.Println()
	table := uitable.New()
	table.MaxColWidth = 80
	table.Wrap = true
	table.AddRow("TASK", "SUBMITTED", "FAILED", "TOTAL")
	table.AddRow(t.TaskID, numMinions-failed, failed, numMinions)
	fmt.Println(table)
}