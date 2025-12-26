package usecase

import (
	"context"
	"fmt"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	usecases_port "kufar-parser-service/internal/core/port/usecases"
	"sync"

	"github.com/google/uuid"
)

type OrchestrateParsingUseCase struct {
	fetchLinksUC usecases_port.FetchLinksPort // <-- Существующий use case для выполнения одной задачи
	reporter port.TaskReporterPort
}

// NewFetchAndEnqueueLinksUseCase создает новый экземпляр FetchAndEnqueueLinksUseCase
func NewOrchestrateParsingUseCase(
	fetchLinksUC usecases_port.FetchLinksPort,
	reporter port.TaskReporterPort,
) *OrchestrateParsingUseCase {
	return &OrchestrateParsingUseCase{
		fetchLinksUC: fetchLinksUC,
		reporter: reporter,
	}
}

func (uc *OrchestrateParsingUseCase) Execute(ctx context.Context, internalTasks []domain.SearchCriteria, taskID uuid.UUID) error {
    
	baseLogger := contextkeys.LoggerFromContext(ctx)
	ucLogger := baseLogger.WithFields(port.Fields{
		"use_case": "OrchestrateParsing",
	})

	ucLogger.Debug("Starting to perform tasks", nil)
    
    
    // 2. Если задач 0 -> отправить отчет
    if len(internalTasks) == 0 {
        ucLogger.Info("DTO translated to zero internal tasks. Nothing to do.", nil)
        // Если задач нет, нужно все равно отчитаться, чтобы кампания завершилась.
        // Отправляем отчет с нулевыми результатами.
        report := &domain.ParsingTasksStats{
			SearchesCompleted: 0,
			NewLinksFound: 0,
        }

		if err := uc.reporter.ReportResults(ctx, taskID, report); err != nil {
			ucLogger.Error("Failed to report task results for zero-task DTO", err, nil)
        }
        
        return nil
    }

	ucLogger.Info("DTO translated to internal search tasks.", port.Fields{"subtasks_count": len(internalTasks)})

   // 2. Запускаем все подзадачи ПАРАЛЛЕЛЬНО и ждем их завершения
  
    var wg sync.WaitGroup
    type subTaskResult struct {
        linksCount int
        err        error
    }
    resultsChan := make(chan subTaskResult, len(internalTasks))

	for _, task := range internalTasks {
		wg.Add(1)
		go func(t domain.SearchCriteria) {
			defer wg.Done()

			subTaskLogger := ucLogger.WithFields(port.Fields{"subtask_name": t.Name})
			taskCtx := contextkeys.ContextWithLogger(ctx, subTaskLogger)
			
			subTaskLogger.Debug("Executing sub-task", nil)
            
            // Execute теперь должен возвращать количество найденных ссылок
			newLinksCount, err := uc.fetchLinksUC.Execute(taskCtx, t, taskID)
			resultsChan <- subTaskResult{linksCount: newLinksCount, err: err}
			if err != nil {
				subTaskLogger.Error("Sub-task failed", err, nil)
			} 
		}(task)
	}
	
	// Блокируемся, пока ВСЕ горутины не вызовут wg.Done()
	wg.Wait()
    close(resultsChan) // Закрываем канал после того, как все горутины завершились
    
    // 3. Агрегируем результаты
    totalNewLinksFound := 0
    successfulSubTasks := 0
	for result := range resultsChan {
        if result.err == nil {
            successfulSubTasks++
        }
        totalNewLinksFound += result.linksCount
    }

	ucLogger.Info("All sub-tasks completed.", port.Fields{
        "total_subtasks": len(internalTasks),
        "successful_subtasks": successfulSubTasks,
        "total_new_links": totalNewLinksFound,
    })

	if successfulSubTasks == 0 && len(internalTasks) > 0 {
        err := fmt.Errorf("all %d sub-tasks failed", len(internalTasks))
        ucLogger.Error("Orchestration failed completely", err, nil)
        
        // Мы НЕ отправляем отчет, а возвращаем ошибку, чтобы RabbitMQ сделал retry.
        return err
    }
    
	finalReport := &domain.ParsingTasksStats{
		SearchesCompleted: 1,
		NewLinksFound: totalNewLinksFound,
	}

	 // `useCase` должен предоставить метод для отправки отчета
    if err := uc.reporter.ReportResults(ctx, taskID, finalReport); err != nil {
		ucLogger.Error("Failed to send final completion report for task", err, nil)
        return err // Возвращаем ошибку, чтобы RabbitMQ попробовал отправить отчет снова
    }
    
    
	return nil
}


