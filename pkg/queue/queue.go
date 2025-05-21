package queue

import (
	"context"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/logger"
)

// Task represents a task to be executed
type Task interface {
	ID() string
	Type() string
	Priority() int
	MaxRetries() int
	Execute(ctx context.Context) error
}

// BaseTask is a basic implementation of Task
type BaseTask struct {
	id         string
	taskType   string
	priority   int
	maxRetries int
	execute    func(ctx context.Context) error
}

// NewBaseTask creates a new BaseTask
func NewBaseTask(id, taskType string, priority, maxRetries int, execute func(ctx context.Context) error) *BaseTask {
	return &BaseTask{
		id:         id,
		taskType:   taskType,
		priority:   priority,
		maxRetries: maxRetries,
		execute:    execute,
	}
}

// ID returns the task ID
func (t *BaseTask) ID() string {
	return t.id
}

// Type returns the task type
func (t *BaseTask) Type() string {
	return t.taskType
}

// Priority returns the task priority
func (t *BaseTask) Priority() int {
	return t.priority
}

// MaxRetries returns the maximum number of retries
func (t *BaseTask) MaxRetries() int {
	return t.maxRetries
}

// Execute executes the task
func (t *BaseTask) Execute(ctx context.Context) error {
	return t.execute(ctx)
}

// TaskQueue represents a queue of tasks
type TaskQueue struct {
	tasks      map[string]Task
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	log        logger.Logger
	numWorkers int
	retryDelay time.Duration
	taskChan   chan Task
}

// NewTaskQueue creates a new TaskQueue
func NewTaskQueue(numWorkers int, retryDelay time.Duration) *TaskQueue {
	ctx, cancel := context.WithCancel(context.Background())

	return &TaskQueue{
		tasks:      make(map[string]Task),
		ctx:        ctx,
		cancel:     cancel,
		log:        logger.GetLogger(),
		numWorkers: numWorkers,
		retryDelay: retryDelay,
		taskChan:   make(chan Task, 100),
	}
}

// Start starts the task queue workers
func (q *TaskQueue) Start() {
	q.log.Infof("Starting task queue with %d workers", q.numWorkers)

	for i := 0; i < q.numWorkers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// Stop stops the task queue
func (q *TaskQueue) Stop() {
	q.log.Info("TaskQueue: Stopping...")
	q.cancel() // Signal workers to stop

	q.log.Debug("TaskQueue: Waiting for workers to finish...")
	q.wg.Wait() // Wait for all workers to complete

	close(q.taskChan) // Close channel after workers are done to avoid panic on send.

	q.log.Info("TaskQueue: Stopped.")
}

// AddTask adds a task to the queue
func (q *TaskQueue) AddTask(task Task) {
	q.mutex.Lock()
	// Check if task already exists
	if _, exists := q.tasks[task.ID()]; exists {
		q.log.Warnf("Task %s already exists, skipping", task.ID())
		q.mutex.Unlock()
		return
	}

	q.log.Debugf("Adding task %s (type: %s, priority: %d)", task.ID(), task.Type(), task.Priority())
	q.tasks[task.ID()] = task
	q.mutex.Unlock() // Unlock before sending to channel to avoid deadlock if channel is full

	// Send task to worker, but respect context cancellation
	select {
	case <-q.ctx.Done():
		q.log.Warnf("TaskQueue: Context done, cannot add task %s", task.ID())
		q.mutex.Lock() // Re-acquire lock to remove task if it was added optimistically
		delete(q.tasks, task.ID())
		q.mutex.Unlock()
		return
	case q.taskChan <- task:
		q.log.Debugf("Task %s sent to worker channel", task.ID())
	default:
		q.log.Warnf("TaskQueue: taskChan is full. Task %s might be delayed or dropped if not handled.", task.ID())
	}
}

// RemoveTask removes a task from the queue
func (q *TaskQueue) RemoveTask(taskID string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if _, exists := q.tasks[taskID]; exists {
		delete(q.tasks, taskID)
		q.log.Debugf("Removed task %s", taskID)
	}
}

// worker processes tasks from the queue
func (q *TaskQueue) worker(workerID int) {
	defer q.wg.Done()
	q.log.Debugf("TaskQueue: Worker %d started", workerID)

	for {
		select {
		case <-q.ctx.Done(): // Prioritize context cancellation
			q.log.Debugf("TaskQueue: Worker %d stopping due to context cancellation.", workerID)
			return
		case task, ok := <-q.taskChan:
			if !ok {
				// taskChan was closed, means queue is shutting down and no more tasks will come.
				q.log.Debugf("TaskQueue: Worker %d stopping because taskChan was closed.", workerID)
				return
			}
			// Double check context before processing, in case of race condition
			select {
			case <-q.ctx.Done():
				q.log.Debugf("TaskQueue: Worker %d received task but context is done. Discarding task %s.", workerID, task.ID())
				// Optionally, re-queue or log discarded task
				continue // Go back to select to exit via ctx.Done() path
			default:
				q.processTask(task, workerID)
			}
		}
	}
}

// processTask processes a single task
func (q *TaskQueue) processTask(task Task, workerID int) {
	q.log.Debugf("TaskQueue: Worker %d processing task %s (type: %s)", workerID, task.ID(), task.Type())

	// Create a task-specific context that respects the queue's main context.
	// This allows individual tasks to have timeouts while still being cancellable by the queue's Stop().
	taskCtx, taskCancel := context.WithTimeout(q.ctx, 5*time.Minute) // Example: 5-minute timeout per task
	defer taskCancel()                                               // Ensure task-specific context is cancelled

	// Execute the task
	var retries int
	var err error

	for retries <= task.MaxRetries() {
		// Check for queue context cancellation before each attempt
		select {
		case <-q.ctx.Done():
			q.log.Debugf("TaskQueue: Worker %d: Queue context cancelled before executing/retrying task %s. Aborting task.", workerID, task.ID())
			return // Exit processing this task
		default:
		}

		if retries > 0 {
			q.log.Debugf("TaskQueue: Worker %d: Retrying task %s (attempt %d/%d)", workerID, task.ID(), retries, task.MaxRetries())
			// Wait before retry, but make the wait cancellable by q.ctx
			select {
			case <-q.ctx.Done():
				q.log.Debugf("TaskQueue: Worker %d: Task %s cancelled during retry delay due to queue context.", workerID, task.ID())
				return // Exit processing this task
			case <-time.After(q.retryDelay):
				// Continue with retry
			}
		}

		err = task.Execute(taskCtx) // Pass the task-specific, cancellable context
		if err == nil {
			q.log.Debugf("TaskQueue: Worker %d: Task %s completed successfully", workerID, task.ID())
			q.RemoveTask(task.ID()) // Remove from the map of active tasks
			return
		}

		// If the task execution was cancelled via its own context (taskCtx), or the queue's context (q.ctx)
		if taskCtx.Err() == context.Canceled || taskCtx.Err() == context.DeadlineExceeded || q.ctx.Err() != nil {
			q.log.Warnf("TaskQueue: Worker %d: Task %s execution cancelled or timed out: taskCtx.Err()=%v, q.ctx.Err()=%v. Original error: %v", workerID, task.ID(), taskCtx.Err(), q.ctx.Err(), err)
			// Do not retry if context was cancelled, as it's an explicit stop signal.
			q.RemoveTask(task.ID())
			return
		}

		q.log.Warnf("TaskQueue: Worker %d: Task %s failed: %v", workerID, task.ID(), err)
		retries++
	}

	q.log.Errorf("TaskQueue: Worker %d: Task %s failed after %d retries: %v", workerID, task.ID(), retries-1, err)
	q.RemoveTask(task.ID()) // Remove from the map of active tasks
}
