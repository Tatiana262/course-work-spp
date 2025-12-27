package usecase

import (
	"context"
	"fmt"
	// "log"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"
	"time"

	"github.com/google/uuid"
)

type ProcessTaskResultUseCase struct {
	repo     port.TaskRepositoryPort
	notifier port.NotifierPort
}

func NewProcessTaskResultUseCase(repo port.TaskRepositoryPort, notifier port.NotifierPort) *ProcessTaskResultUseCase {
	return &ProcessTaskResultUseCase{
		repo:     repo,
		notifier: notifier,
	}
}



func (uc *ProcessTaskResultUseCase) Execute(ctx context.Context, taskID uuid.UUID, results map[string]int) error {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "ProcessTaskResult",
		// "task_id":  taskID.String(),
	})
	
	ucLogger.Info("Use case started: processing result for task", nil)

	task, err := uc.repo.FindByID(ctx, taskID)
	if err != nil {
		ucLogger.Error("Repository failed to find task", err, nil)
		return err // Если задача не найдена, RabbitMQ повторит попытку
	}

	if task.Status == domain.StatusCompleted || task.Status == domain.StatusFailed { //
		ucLogger.Warn("Task is already failed or completed, skipping updating task summary", nil)
		return nil
	}

	// Атомарно инкрементируем счетчики в БД
	updatedTask, err := uc.repo.IncrementSummary(ctx, taskID, results)
	if err != nil {
		ucLogger.Error("Repository failed to increment summary", err, nil)
		return err
	}
	
	
	// Проверяем, не пора ли завершать задачу
	summary := updatedTask.ResultSummary
	
	shouldComplete := false
	logMessage := ""
	
	expectedCount := getIntFromSummary(summary, "expected_results_count")

	// пришла команда с информацией о завершении
	if _, ok := results["expected_results_count"]; ok && expectedCount > 0 {
		ucLogger.Info("Completion command processed, checking if task can be completed now.", nil)
	}
	

	if updatedTask.Type == "FIND_NEW" {
		searchesCompleted := getIntFromSummary(summary, "searches_completed")
		newLinksFound := getIntFromSummary(summary, "new_links_found")
		totalProcessed := getIntFromSummary(summary, "total_processed")

		// Условие завершения для FIND_NEW:
		if expectedCount > 0 && searchesCompleted >= expectedCount && totalProcessed >= newLinksFound {
			shouldComplete = true
			logMessage = fmt.Sprintf("All searches completed (%d/%d) and all links processed (%d/%d)", searchesCompleted, expectedCount, totalProcessed, newLinksFound)
		}
	} else { // Для 'ACTUALIZE_ACTIVE', 'ACTUALIZE_ARCHIVED'
		totalProcessed := getIntFromSummary(summary, "total_processed")
		
		if expectedCount > 0 && totalProcessed >= expectedCount {
			shouldComplete = true
			logMessage = fmt.Sprintf("All results received (%d/%d)", totalProcessed, expectedCount)
		}
	}

	// Если нужно, завершаем задачу
	if shouldComplete {
		uc.markTaskAsCompleted(ctx, updatedTask, logMessage)
	}

	ucLogger.Debug("Notifying clients about task progress", nil)
	// Отправляем уведомление о текущем состоянии
	uc.notifier.Notify(ctx, port.TaskEvent{Type: "task_updated", Data: *updatedTask})
	ucLogger.Info("Use case finished successfully", port.Fields{
		"task_status": updatedTask.Status,
	})
	
	return nil
}

func getIntFromSummary(summary domain.ResultSummary, key string) int {
	if val, ok := summary[key].(float64); ok {
		return int(val)
	}
	return 0
}


func (uc *ProcessTaskResultUseCase) markTaskAsCompleted(ctx context.Context, task *domain.Task, logMessage string) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "ProcessTaskResult.markTaskAsCompleted",
		"task_id":  task.ID.String(),
	})

	ucLogger.Info("Marking task as completed", port.Fields{"reason": logMessage})
	task.Status = domain.StatusCompleted
	now := time.Now().UTC()
	task.FinishedAt = &now
	
	if err := uc.repo.Update(ctx, task); err != nil {
		ucLogger.Error("Repository failed to mark task as completed", err, nil)
	}
}





// func (uc *ProcessTaskResultUseCase) Execute(ctx context.Context, taskID uuid.UUID, results map[string]int) error {
// 	logger := contextkeys.LoggerFromContext(ctx)
// 	ucLogger := logger.WithFields(port.Fields{
// 		"use_case": "ProcessTaskResult",
// 		"task_id":  taskID.String(),
// 	})
	
// 	ucLogger.Info("Use case started: processing incoming result", port.Fields{"results": results})
	
// 	// Используем специальный метод репозитория для атомарного обновления
// 	if err := uc.repo.IncrementSummary(ctx, taskID, results); err != nil {
// 		ucLogger.Error("Repository failed to increment summary", err, nil)
// 		return err
// 	}
// 	ucLogger.Debug("Summary incremented successfully", nil)

// 	// log.Println("!!!!!!!!!!!!!!!!!!")
// 	// log.Println(results)

// 	// После обновления, получаем актуальное состояние задачи, чтобы отправить его подписчикам
// 	updatedTask, err := uc.repo.FindByID(ctx, taskID)
// 	if err != nil {
// 		// Даже если не удалось получить, ошибка инкремента не произошла, так что не возвращаем ее
// 		ucLogger.Error("Could not fetch task after incrementing summary, further processing stopped", err, nil)
// 		return nil
// 	}

// 	if updatedTask.Status == domain.StatusCompleted || updatedTask.Status == domain.StatusFailed {
// 		ucLogger.Debug("Task is already in a final state, no further action needed.", port.Fields{"status": updatedTask.Status})
// 		return nil
// 	}

// 	summary := updatedTask.ResultSummary
// 	expectedCount := getIntFromSummary(summary, "expected_results_count")

// 	// log.Println("!!!!!!!!!!!!!!!!!!")
// 	// log.Println(summary)
// 	ucLogger.Debug("Checking task completion status after increment.", port.Fields{"summary": updatedTask.ResultSummary})

// 	if updatedTask.Type != "FIND_NEW" {

// 		totalProcessed := getIntFromSummary(summary, "total_processed")

// 		// ---> Логика завершения <---
// 		// Условие: мы знаем, сколько ждать (expected > 0), текущий статус еще не "completed",
// 		// и мы достигли или превысили ожидаемое количество.
// 		if expectedCount > 0 && totalProcessed >= expectedCount {
// 			uc.markTaskAsCompleted(ctx, updatedTask, fmt.Sprintf("Final result received (%d/%d)", totalProcessed, expectedCount))
// 		}

// 	} else {

// 		searchesCompleted := getIntFromSummary(summary, "searches_completed")
// 		newLinksFound := getIntFromSummary(summary, "new_links_found")
// 		totalProcessed := getIntFromSummary(summary, "total_processed") // `created`


// 		// и мы достигли или превысили ожидаемое количество.
// 		if expectedCount > 0 && searchesCompleted >= expectedCount && totalProcessed >= newLinksFound && updatedTask.Status != domain.StatusCompleted {
// 			uc.markTaskAsCompleted(ctx, updatedTask, fmt.Sprintf("All searches completed (%d/%d) and all links processed (%d/%d)", searchesCompleted, expectedCount, totalProcessed, newLinksFound))
// 		}

// 	}

// 	ucLogger.Debug("Notifying clients about task progress", nil)
// 	// Отправляем уведомление о прогрессе
// 	uc.notifier.Notify(ctx, port.TaskEvent{Type: "task_updated", Data: *updatedTask})
// 	ucLogger.Info("Use case finished successfully", port.Fields{
// 		"task_status": updatedTask.Status,
// 	})

// 	return nil
// }