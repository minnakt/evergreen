/*
Task Output Directory

The task output directory establishes an API, via the file system, for the
handling of output generated by an Evergreen task. It sits on the top level of
the task's working directory and contains subdirectories that map to specific
types of task output. Any data written to these subdirectories is automatically
ingested, processed, and persisted in a well defined manner established with
with users. A subdirectory may require a specification file written by the task
workload detailing versions, formats, etc. In general, a task output
subdirectory is free to establish its own requirements, conventions, and
behavior.

This package implements the task output directory API for the Evergreen agent.
*/
package taskoutput

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/evergreen-ci/evergreen/agent/internal/client"
	"github.com/evergreen-ci/evergreen/agent/internal/redactor"
	"github.com/evergreen-ci/evergreen/model/task"
	"github.com/evergreen-ci/evergreen/taskoutput"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/recovery"
	"github.com/pkg/errors"
)

var directoryHandlerFactories = map[string]directoryHandlerFactory{
	"TestLogs": newTestLogDirectoryHandler,
}

// Directory is the application representation of a task's reserved output
// directory. It coordinates the automated handling of task output written to
// the reserved directory while a task runs.
type Directory struct {
	root     string
	handlers map[string]directoryHandler
}

// NewDirectory returns a new task output directory with the specified root for
// the given task.
func NewDirectory(root string, tsk *task.Task, redactorOpts redactor.RedactionOptions, logger client.LoggerProducer) *Directory {
	output := tsk.TaskOutputInfo
	taskOpts := taskoutput.TaskOptions{
		ProjectID: tsk.Project,
		TaskID:    tsk.Id,
		Execution: tsk.Execution,
	}

	root = filepath.Join(root, "build")
	handlers := map[string]directoryHandler{}
	for name, factory := range directoryHandlerFactories {
		dir := filepath.Join(root, name)
		handlers[dir] = factory(dir, output, taskOpts, redactorOpts, logger)
	}

	return &Directory{
		root:     root,
		handlers: handlers,
	}
}

// Setup creates the subdirectories for each handler and should be called
// during task set up.
func (d *Directory) Setup() error {
	for dir := range d.handlers {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return errors.Wrapf(err, "creating task output directory '%s'", dir)
		}
	}

	return nil
}

// Run runs each directory handler and removes the task output directory.
func (d *Directory) Run(ctx context.Context) error {
	catcher := grip.NewBasicCatcher()

	var wg sync.WaitGroup
	for id, handler := range d.handlers {
		wg.Add(1)
		go func(id string, h directoryHandler) {
			defer func() {
				catcher.Add(recovery.HandlePanicWithError(recover(), nil, "task output directory handler runner"))
			}()
			defer wg.Done()

			catcher.Wrapf(h.run(ctx), "running task output for '%s'", id)
		}(id, handler)
	}
	wg.Wait()

	catcher.Wrap(os.RemoveAll(d.root), "removing task output directory")

	return catcher.Resolve()
}

// directoryHandler abstracts the automatic handling of task output for
// individual subdirectories.
type directoryHandler interface {
	run(ctx context.Context) error
}

// directoryHandlerFactory abstracts the creation of a directory handler.
type directoryHandlerFactory func(string, *taskoutput.TaskOutput, taskoutput.TaskOptions, redactor.RedactionOptions, client.LoggerProducer) directoryHandler
