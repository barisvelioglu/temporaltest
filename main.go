package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART samples-go-child-workflow-example-worker-starter
func main() {
	// The client is a heavyweight object that should be created only once per process.
	c, err := client.Dial(client.Options{
		HostPort:  client.DefaultHostPort,
		Namespace: "SimpleDomain",
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "child-workflow", worker.Options{
		MaxConcurrentActivityExecutionSize: 10,
	})

	// namespaceClient, err := client.NewNamespaceClient(client.Options{
	// 	HostPort: client.DefaultHostPort,
	// })
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// duration := time.Duration(time.Hour * 24 * 3)
	// err = namespaceClient.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
	// 	Namespace:                        "SimpleDomain",
	// 	Description:                      "Maestro Workflows",
	// 	OwnerEmail:                       "baris.velioglu@maestrohub.com",
	// 	WorkflowExecutionRetentionPeriod: &duration,
	// })
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	w.RegisterWorkflow(SampleParentWorkflow)
	w.RegisterWorkflow(SampleChildWorkflow)
	w.RegisterWorkflow(SampleChildWorkflow2)
	w.RegisterActivity(SampleActivity)
	w.RegisterActivity(SampleActivity2)

	bas := false
	if bas {
		workflowOptions := client.StartWorkflowOptions{
			TaskQueue:                "child-workflow",
			WorkflowExecutionTimeout: time.Second * 120,
			WorkflowRunTimeout:       time.Second * 120,
			WorkflowTaskTimeout:      time.Second * 5,
		}

		// go func() {
		// 	for {

		// 		workflowRun, err := c.ExecuteWorkflow(context.Background(), workflowOptions, SampleParentWorkflow)
		// 		if err != nil {
		// 			log.Fatalln("Unable to execute workflow", err)
		// 		}
		// 		log.Println("Started workflow",
		// 			"WorkflowID", workflowRun.GetID(), "RunID", workflowRun.GetRunID())

		// 		// Synchronously wait for the Workflow Execution to complete.
		// 		// Behind the scenes the SDK performs a long poll operation.
		// 		// If you need to wait for the Workflow Execution to complete from another process use
		// 		// Client.GetWorkflow API to get an instance of the WorkflowRun.
		// 		var result string
		// 		err = workflowRun.Get(context.Background(), &result)
		// 		if err != nil {
		// 			log.Fatalln("Failure getting workflow result", err)
		// 		}
		// 		log.Printf("Workflow result: %v", result)

		// 		log.Println(result)
		// 	}
		// }()

		go func() {

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

		}()
	}

	go func() {
		e := echo.New()
		e.GET("/", func(c echo.Context) error {
			return c.String(http.StatusOK, "Hello, World!")
		})
		e.Logger.Fatal(e.Start(":31323"))
	}()

	defer w.Stop()
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}

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

	logger.Info("Parent execution completed.", "Result", result)
	return result, nil
}

// @@@SNIPSTART samples-go-child-workflow-example-child-workflow-definition
// SampleChildWorkflow is a Workflow Definition
func SampleChildWorkflow(ctx workflow.Context, name string) (string, error) {
	logger := workflow.GetLogger(ctx)
	greeting := "Hello " + name + "!"
	logger.Info("Child workflow execution: " + greeting)

	activityOptions := workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Second * 15, //ActivityTaskScheduled ---> ActivityTaskCompleted
		ScheduleToStartTimeout: time.Second * 15, //ActivityTaskScheduled ---> ActivityTaskStarted
		StartToCloseTimeout:    time.Second * 15, //ActivityTaskStarted ---> ActivityTaskCompleted
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	results := []string{}

	var routineError error
	var blabla int
	for i := 0; i < 10; i++ {
		x := i
		workflow.Go(ctx, func(ctx workflow.Context) {
			var result string
			routineError = workflow.ExecuteActivity(ctx, SampleActivity2, fmt.Sprintf("%d", x)).Get(ctx, &result)
			blabla++
			results = append(results, result)
		})
	}

	for i := 0; i < 10; i++ {
		var result string
		workflow.ExecuteActivity(ctx, SampleActivity, fmt.Sprintf("%d", i)).Get(ctx, &result)

		results = append(results, result)
	}

	_ = workflow.Await(ctx, func() bool {
		return routineError != nil || blabla == 10
	})

	return strings.Join(results, "||"), nil
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
	err := workflow.ExecuteActivity(ctx, SampleActivity, name).Get(ctx, &result)
	if err != nil {
		return result, err
	}
	logger.Info("Child workflow execution: " + result)
	return result, nil
}

// @@@SNIPEND

func SampleActivity(ctx context.Context, name string) (string, error) {
	time.Sleep(time.Second * 1)
	return "Hello baby, " + name, nil
}

func SampleActivity2(ctx context.Context, name string) (string, error) {
	time.Sleep(time.Second * 5)
	return "Bye baby, " + name, nil
}
