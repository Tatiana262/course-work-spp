package usecase

import (
	"context"
	// "log"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"
	"time"

	"github.com/google/uuid"
)

// CompleteTaskUseCase реализует порт CompleteTaskUseCasePort.
type CompleteTaskUseCase struct {
	repo     port.TaskRepositoryPort
	notifier port.NotifierPort
}

func NewCompleteTaskUseCase(repo port.TaskRepositoryPort, notifier port.NotifierPort) *CompleteTaskUseCase {
	return &CompleteTaskUseCase{repo: repo, notifier: notifier}
}

func (uc *CompleteTaskUseCase) Execute(ctx context.Context, taskID uuid.UUID, expectedCount int) error {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "CompleteTask",
		"task_id":  taskID.String(),
		"expected_count": expectedCount,
	})

	ucLogger.Info("Use case started: processing completion command", nil)
	
	task, err := uc.repo.FindByID(ctx, taskID)
	if err != nil {
		ucLogger.Error("Repository failed to find task", err, nil)
		return err // Если задача не найдена, RabbitMQ повторит попытку, это нормально.
	}

	if task.Status == domain.StatusFailed { //task.Status == domain.StatusCompleted ||
		ucLogger.Warn("Task is already failed, skipping completion logic.", nil)
		return nil
	}

	// Получаем текущее количество обработанных результатов
	currentProcessed := 0
	if val, ok := task.ResultSummary["total_processed"].(float64); ok {
		currentProcessed = int(val)
	}
	
	// ---> Логика <---
	// 1. Записываем, сколько результатов мы ожидаем
	if task.ResultSummary == nil {
		task.ResultSummary = make(domain.ResultSummary)
	}
	task.ResultSummary["expected_results_count"] = expectedCount
	
	// 2. Проверяем, не завершена ли задача УЖЕ
	if currentProcessed >= expectedCount {
		ucLogger.Info("Completion condition met. Marking task as completed.", port.Fields{"current_processed": currentProcessed})
		task.Status = domain.StatusCompleted
		now := time.Now().UTC()
		task.FinishedAt = &now
	} else {
		ucLogger.Info("Completion condition not yet met.", port.Fields{"current_processed": currentProcessed})
	}

	// Сохраняем изменения (либо только `expected_count`, либо еще и статус)
	if err := uc.repo.Update(ctx, task); err != nil {
		ucLogger.Error("Repository failed to update task", err, nil)
		return err
	}
	
	ucLogger.Info("Task updated, notifying clients", nil)
	// Уведомляем клиентов об обновлении
	uc.notifier.Notify(ctx, port.TaskEvent{Type: "task_updated", Data: *task})
	ucLogger.Info("Use case finished successfully", nil)

	return nil
}