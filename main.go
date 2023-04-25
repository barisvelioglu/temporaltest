package main

import (
	"context"
	"log"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART samples-go-child-workflow-example-worker-starter
func main() {
	// The client is a heavyweight object that should be created only once per process.
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "child-workflow", worker.Options{
		MaxConcurrentActivityExecutionSize: 10,
	})

	w.RegisterWorkflow(SampleParentWorkflow)
	w.RegisterWorkflow(SampleChildWorkflow)
	w.RegisterWorkflow(SampleChildWorkflow2)
	w.RegisterActivity(Activity)

	go func() {
		err = w.Run(worker.InterruptCh())
		if err != nil {
			log.Fatalln("Unable to start worker", err)
		}

	}()
	defer w.Stop()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue:                "child-workflow",
		WorkflowExecutionTimeout: time.Second * 10,
		WorkflowRunTimeout:       time.Second * 10,
		WorkflowTaskTimeout:      time.Second * 5,
	}

	go func() {
		for {

			workflowRun, err := c.ExecuteWorkflow(context.Background(), workflowOptions, SampleParentWorkflow)
			if err != nil {
				log.Fatalln("Unable to execute workflow", err)
			}
			log.Println("Started workflow",
				"WorkflowID", workflowRun.GetID(), "RunID", workflowRun.GetRunID())

			// Synchronously wait for the Workflow Execution to complete.
			// Behind the scenes the SDK performs a long poll operation.
			// If you need to wait for the Workflow Execution to complete from another process use
			// Client.GetWorkflow API to get an instance of the WorkflowRun.
			var result string
			err = workflowRun.Get(context.Background(), &result)
			if err != nil {
				log.Fatalln("Failure getting workflow result", err)
			}
			log.Printf("Workflow result: %v", result)

			log.Println(result)
		}
	}()

	select {}

}

// @@@SNIPSTART samples-go-child-workflow-example-parent-workflow-definition
// SampleParentWorkflow is a Workflow Definition
// This Workflow Definition demonstrates how to start a Child Workflow Execution from a Parent Workflow Execution.
// Each Child Workflow Execution starts a new Run.
// The Parent Workflow Execution is notified only after the completion of last Run of the Child Workflow Execution.
func SampleParentWorkflow(ctx workflow.Context) (string, error) {
	logger := workflow.GetLogger(ctx)

	cwo := workflow.ChildWorkflowOptions{}
	ctx = workflow.WithChildOptions(ctx, cwo)

	var result string
	err := workflow.ExecuteChildWorkflow(ctx, SampleChildWorkflow, "World").Get(ctx, &result)
	if err != nil {
		logger.Error("Parent execution received child execution failure.", "Error", err)
		return "", err
	}

	err = workflow.ExecuteChildWorkflow(ctx, SampleChildWorkflow2, "World").Get(ctx, &result)
	if err != nil {
		logger.Error("Parent execution received child execution failure.", "Error", err)
		return "", err
	}

	logger.Info("Parent execution completed.", "Result", result)
	return result, nil
}

// @@@SNIPSTART samples-go-child-workflow-example-child-workflow-definition
// SampleChildWorkflow is a Workflow Definition
func SampleChildWorkflow(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	greeting := "Hello " + name + "!"
	logger.Info("Child workflow execution: " + greeting)
	return greeting, nil
}

// @@@SNIPEND

// @@@SNIPSTART samples-go-child-workflow-example-child-workflow-definition
// SampleChildWorkflow is a Workflow Definition
func SampleChildWorkflow2(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	var result string

	activityoptions := workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Second * 5,
		ScheduleToStartTimeout: time.Second * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityoptions)
	err := workflow.ExecuteActivity(ctx, Activity, name).Get(ctx, &result)
	if err != nil {
		return result, err
	}
	logger.Info("Child workflow execution: " + result)
	return result, nil
}

// @@@SNIPEND

func Activity(ctx context.Context, name string) (string, error) {
	return "Hello baby, " + name, nil
}
