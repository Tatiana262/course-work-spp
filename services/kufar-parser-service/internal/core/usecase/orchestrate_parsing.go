package usecase

import (
	"context"
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

	ucLogger.Info("Starting to perform tasks", nil)
    
    
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

	ucLogger.Info("Translated DTO to internal search tasks.", port.Fields{"subtasks_count": len(internalTasks)})

   // 2. Запускаем все подзадачи ПАРАЛЛЕЛЬНО и ждем их завершения
	var wg sync.WaitGroup
    // Создаем канал для сбора статистики из горутин
    statsChan := make(chan int, len(internalTasks))

	for _, task := range internalTasks {
		wg.Add(1)
		go func(t domain.SearchCriteria) {
			defer wg.Done()

			subTaskLogger := ucLogger.WithFields(port.Fields{"subtask_name": t.Name})
			taskCtx := contextkeys.ContextWithLogger(ctx, subTaskLogger)
			
			subTaskLogger.Info("Executing sub-task", nil)
            
            // Execute теперь должен возвращать количество найденных ссылок
			newLinksCount, err := uc.fetchLinksUC.Execute(taskCtx, t, taskID)
			if err != nil {
				subTaskLogger.Error("Sub-task failed", err, nil)
                statsChan <- 0 // Отправляем 0, если была ошибка
			} else {
                statsChan <- newLinksCount // Отправляем результат в канал
            }
		}(task)
	}
	
	// Блокируемся, пока ВСЕ горутины не вызовут wg.Done()
	wg.Wait()
    close(statsChan) // Закрываем канал после того, как все горутины завершились
    
    // 3. Агрегируем результаты
    totalNewLinksFound := 0
    for count := range statsChan {
        totalNewLinksFound += count
    }

	ucLogger.Info("All sub-tasks completed.", port.Fields{"total_new_links": totalNewLinksFound})
    
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


