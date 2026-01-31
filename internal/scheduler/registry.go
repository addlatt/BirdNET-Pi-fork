package scheduler

import (
	"fmt"
	"sync"
)

// Registry manages task registration and lookup.
type Registry struct {
	tasks map[string]Task
	mu    sync.RWMutex
}

// NewRegistry creates a new task registry.
func NewRegistry() *Registry {
	return &Registry{
		tasks: make(map[string]Task),
	}
}

// Register adds a task to the registry.
// Returns an error if a task with the same name already exists.
func (r *Registry) Register(task Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := task.Name()
	if _, exists := r.tasks[name]; exists {
		return fmt.Errorf("task %q already registered", name)
	}

	r.tasks[name] = task
	return nil
}

// MustRegister adds a task to the registry and panics on error.
// Useful for registering tasks at startup.
func (r *Registry) MustRegister(task Task) {
	if err := r.Register(task); err != nil {
		panic(err)
	}
}

// Get returns a task by name, or nil if not found.
func (r *Registry) Get(name string) Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tasks[name]
}

// List returns all registered tasks.
func (r *Registry) List() []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Names returns the names of all registered tasks.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tasks))
	for name := range r.tasks {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered tasks.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tasks)
}
