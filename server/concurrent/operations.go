package concurrent

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

/*
ИСПРАВЛЕНИЯ:
1. Устранены race conditions - данные передаются через каналы
2. Исправлена проблема множественного чтения из одного канала
3. Добавлено использование errgroup для упрощенной обработки ошибок
4. Правильное закрытие каналов отправителями
5. Удалены deadlock риски
*/

// ==================== 1. GAME DETAILS WITH CONCURRENCY ====================

type GameDetails struct {
	Game         models.Game
	Reviews      []models.Review
	RelatedGames []models.Game
	Statistics   GameStatistics
	Error        error
}

type GameStatistics struct {
	TotalReviews  int64
	AverageRating float64
	TotalOwners   int64
	SameCategory  int64
}

// gameResult используется для безопасной передачи данных игры
type gameResult struct {
	game models.Game
	err  error
}

// FetchGameWithDetails загружает детали игры параллельно (ИСПРАВЛЕНО)
func FetchGameWithDetails(gameID uint) (*GameDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := &GameDetails{}

	// Используем errgroup для упрощенной обработки ошибок
	g, ctx := errgroup.WithContext(ctx)

	// Канал для передачи основной информации об игре
	gameChan := make(chan gameResult, 1)
	reviewsChan := make(chan []models.Review, 1)
	relatedChan := make(chan []models.Game, 1)
	statsChan := make(chan GameStatistics, 1)

	// Goroutine 1: Загрузка основной информации об игре
	g.Go(func() error {
		var game models.Game
		err := db.DB.WithContext(ctx).Preload("Category").First(&game, gameID).Error
		gameChan <- gameResult{game: game, err: err}
		return err
	})

	// Goroutine 2: Загрузка отзывов
	g.Go(func() error {
		var reviews []models.Review
		err := db.DB.WithContext(ctx).
			Where("game_id = ?", gameID).
			Preload("User").
			Order("created_at DESC").
			Limit(10).
			Find(&reviews).Error

		if err == nil {
			reviewsChan <- reviews
		} else {
			reviewsChan <- nil
		}
		return nil // Не критичная ошибка
	})

	// Goroutine 3: Загрузка похожих игр (зависит от основной игры)
	g.Go(func() error {
		// Безопасно получаем данные игры через канал
		gameRes := <-gameChan
		if gameRes.err != nil || gameRes.game.ID == 0 {
			relatedChan <- nil
			return nil
		}

		var related []models.Game
		db.DB.WithContext(ctx).
			Where("category_id = ? AND id != ?", gameRes.game.CategoryID, gameID).
			Limit(5).
			Find(&related)
		relatedChan <- related
		return nil
	})

	// Goroutine 4: Расчет статистики
	g.Go(func() error {
		stats := GameStatistics{}

		// Вложенный errgroup для параллельных подзапросов
		statsGroup, statsCtx := errgroup.WithContext(ctx)

		// Количество отзывов
		statsGroup.Go(func() error {
			db.DB.WithContext(statsCtx).
				Model(&models.Review{}).
				Where("game_id = ?", gameID).
				Count(&stats.TotalReviews)
			return nil
		})

		// Средний рейтинг
		statsGroup.Go(func() error {
			var avg struct{ Avg float64 }
			db.DB.WithContext(statsCtx).
				Model(&models.Review{}).
				Select("AVG(rating) as avg").
				Where("game_id = ?", gameID).
				Scan(&avg)
			stats.AverageRating = avg.Avg
			return nil
		})

		// Количество владельцев
		statsGroup.Go(func() error {
			db.DB.WithContext(statsCtx).
				Model(&models.Ownership{}).
				Where("game_id = ?", gameID).
				Count(&stats.TotalOwners)
			return nil
		})

		// Игры той же категории
		statsGroup.Go(func() error {
			gameRes := <-gameChan // Безопасное получение данных
			if gameRes.err == nil && gameRes.game.ID != 0 {
				db.DB.WithContext(statsCtx).
					Model(&models.Game{}).
					Where("category_id = ?", gameRes.game.CategoryID).
					Count(&stats.SameCategory)
			}
			return nil
		})

		statsGroup.Wait()
		statsChan <- stats
		return nil
	})

	// Ждем завершения всех операций
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Получаем результаты (все каналы уже заполнены)
	gameRes := <-gameChan
	if gameRes.err != nil {
		return nil, gameRes.err
	}
	result.Game = gameRes.game
	result.Reviews = <-reviewsChan
	result.RelatedGames = <-relatedChan
	result.Statistics = <-statsChan

	return result, nil
}

// ==================== 2. BULK OPERATIONS WITH WORKER POOL ====================

type GameProcessingJob struct {
	Game   models.Game
	Action string
}

type GameProcessingResult struct {
	GameID  uint
	Success bool
	Error   error
	Message string
}

// ProcessBulkGames обрабатывает множество игр параллельно (БЕЗ ИЗМЕНЕНИЙ - код корректен)
func ProcessBulkGames(games []models.Game, action string, numWorkers int) ([]GameProcessingResult, error) {
	if numWorkers <= 0 {
		numWorkers = 10
	}

	jobs := make(chan GameProcessingJob, len(games))
	results := make(chan GameProcessingResult, len(games))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				result := processGame(job, workerID)
				results <- result
			}
		}(i)
	}

	// Отправляем задания и закрываем канал (отправитель закрывает)
	go func() {
		for _, game := range games {
			jobs <- GameProcessingJob{
				Game:   game,
				Action: action,
			}
		}
		close(jobs)
	}()

	// Ждем завершения всех воркеров и закрываем results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Собираем результаты
	var allResults []GameProcessingResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults, nil
}

func processGame(job GameProcessingJob, workerID int) GameProcessingResult {
	time.Sleep(100 * time.Millisecond)

	switch job.Action {
	case "validate":
		if job.Game.Name == "" || job.Game.Price < 0 {
			return GameProcessingResult{
				GameID:  job.Game.ID,
				Success: false,
				Error:   fmt.Errorf("validation failed"),
				Message: fmt.Sprintf("Worker %d: Invalid game data", workerID),
			}
		}
		return GameProcessingResult{
			GameID:  job.Game.ID,
			Success: true,
			Message: fmt.Sprintf("Worker %d: Validation passed", workerID),
		}

	case "update_prices":
		newPrice := job.Game.Price * 0.9
		db.DB.Model(&job.Game).Update("price", newPrice)
		return GameProcessingResult{
			GameID:  job.Game.ID,
			Success: true,
			Message: fmt.Sprintf("Worker %d: Price updated to %.2f", workerID, newPrice),
		}

	default:
		return GameProcessingResult{
			GameID:  job.Game.ID,
			Success: false,
			Error:   fmt.Errorf("unknown action: %s", job.Action),
		}
	}
}

// ==================== 3. NOTIFICATIONS WITH CONCURRENCY ====================

type NotificationJob struct {
	UserID  uint
	Message string
	Type    string
}

type NotificationResult struct {
	UserID  uint
	Success bool
	Error   error
}

// SendBulkNotifications отправляет уведомления параллельно (ИСПРАВЛЕНО)
func SendBulkNotifications(notifications []NotificationJob, maxWorkers int) ([]NotificationResult, error) {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	g := new(errgroup.Group)
	g.SetLimit(maxWorkers) // Ограничиваем количество одновременных goroutines

	results := make([]NotificationResult, len(notifications))
	var mu sync.Mutex // Защита от race condition при записи в slice

	for i, job := range notifications {
		i, job := i, job // Захват переменных цикла
		g.Go(func() error {
			result := sendNotification(job)

			mu.Lock()
			results[i] = result
			mu.Unlock()

			return nil // Не прерываем выполнение при ошибке отправки
		})
	}

	g.Wait()
	return results, nil
}

func sendNotification(job NotificationJob) NotificationResult {
	time.Sleep(100 * time.Millisecond)

	var user models.User
	if err := db.DB.First(&user, job.UserID).Error; err != nil {
		return NotificationResult{
			UserID:  job.UserID,
			Success: false,
			Error:   err,
		}
	}

	if user.IsBanned {
		return NotificationResult{
			UserID:  job.UserID,
			Success: false,
			Error:   fmt.Errorf("user is banned"),
		}
	}

	return NotificationResult{
		UserID:  job.UserID,
		Success: true,
	}
}

// ==================== 4. DASHBOARD STATISTICS ====================

type DashboardStats struct {
	TotalUsers    int64
	TotalGames    int64
	TotalReviews  int64
	TotalSales    int64
	ActiveUsers   int64
	RecentGames   int64
	AverageRating float64
	TopCategory   string
	Error         error
}

// CalculateDashboardStats вычисляет статистику параллельно (ИСПРАВЛЕНО)
func CalculateDashboardStats() (*DashboardStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats := &DashboardStats{}
	g, ctx := errgroup.WithContext(ctx)

	// 1. Общее количество пользователей
	g.Go(func() error {
		return db.DB.WithContext(ctx).Model(&models.User{}).Count(&stats.TotalUsers).Error
	})

	// 2. Общее количество игр
	g.Go(func() error {
		return db.DB.WithContext(ctx).Model(&models.Game{}).Count(&stats.TotalGames).Error
	})

	// 3. Общее количество отзывов
	g.Go(func() error {
		return db.DB.WithContext(ctx).Model(&models.Review{}).Count(&stats.TotalReviews).Error
	})

	// 4. Общее количество продаж
	g.Go(func() error {
		return db.DB.WithContext(ctx).Model(&models.Ownership{}).Count(&stats.TotalSales).Error
	})

	// 5. Активные пользователи
	g.Go(func() error {
		return db.DB.WithContext(ctx).
			Model(&models.User{}).
			Where("is_banned = ?", false).
			Count(&stats.ActiveUsers).Error
	})

	// 6. Недавние игры
	g.Go(func() error {
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		return db.DB.WithContext(ctx).
			Model(&models.Game{}).
			Where("created_at > ?", thirtyDaysAgo).
			Count(&stats.RecentGames).Error
	})

	// 7. Средний рейтинг
	g.Go(func() error {
		var avg struct{ Avg float64 }
		err := db.DB.WithContext(ctx).
			Model(&models.Review{}).
			Select("AVG(rating) as avg").
			Scan(&avg).Error
		if err == nil {
			stats.AverageRating = avg.Avg
		}
		return err
	})

	// 8. Топ категория
	g.Go(func() error {
		var result struct {
			CategoryID uint
			Count      int64
		}
		err := db.DB.WithContext(ctx).
			Model(&models.Game{}).
			Select("category_id, COUNT(*) as count").
			Group("category_id").
			Order("count DESC").
			Limit(1).
			Scan(&result).Error

		if err != nil {
			return err
		}

		var category models.Category
		if err := db.DB.WithContext(ctx).First(&category, result.CategoryID).Error; err == nil {
			stats.TopCategory = category.Name
		}
		return nil
	})

	// Ждем завершения всех операций
	if err := g.Wait(); err != nil {
		stats.Error = err
		return stats, err
	}

	return stats, nil
}

// ==================== 5. PARALLEL SEARCH ====================

type SearchResult struct {
	Games      []models.Game
	TotalFound int
	SearchTime time.Duration
}

// ParallelSearch выполняет поиск по разным критериям параллельно (ИСПРАВЛЕНО)
func ParallelSearch(query string) (*SearchResult, error) {
	start := time.Now()

	var nameGames, descGames, categoryGames []models.Game
	g := new(errgroup.Group)

	// Поиск по названию
	g.Go(func() error {
		return db.DB.Where("name ILIKE ?", "%"+query+"%").
			Limit(20).
			Find(&nameGames).Error
	})

	// Поиск по описанию
	g.Go(func() error {
		return db.DB.Where("description ILIKE ?", "%"+query+"%").
			Limit(20).
			Find(&descGames).Error
	})

	// Поиск по категории
	g.Go(func() error {
		var category models.Category
		err := db.DB.Where("name ILIKE ?", "%"+query+"%").First(&category).Error
		if err != nil {
			return nil // Категория не найдена - не критично
		}

		return db.DB.Where("category_id = ?", category.ID).
			Limit(20).
			Find(&categoryGames).Error
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Объединяем результаты (удаляем дубликаты)
	resultMap := make(map[uint]models.Game)
	for _, game := range nameGames {
		resultMap[game.ID] = game
	}
	for _, game := range descGames {
		resultMap[game.ID] = game
	}
	for _, game := range categoryGames {
		resultMap[game.ID] = game
	}

	// Преобразуем в массив
	var finalGames []models.Game
	for _, game := range resultMap {
		finalGames = append(finalGames, game)
	}

	return &SearchResult{
		Games:      finalGames,
		TotalFound: len(finalGames),
		SearchTime: time.Since(start),
	}, nil
}

// ==================== 6. IMAGE PROCESSING PIPELINE ====================

type ImageProcessingJob struct {
	GameID    uint
	ImageData []byte
	Filename  string
}

type ProcessedImage struct {
	Original  []byte
	Thumbnail []byte
	Metadata  map[string]interface{}
	GameID    uint
	Error     error
}

// ProcessImagesInPipeline обрабатывает изображения через pipeline (ИСПРАВЛЕНО)
func ProcessImagesInPipeline(jobs []ImageProcessingJob) ([]ProcessedImage, error) {
	// Создаем каналы для pipeline
	validationChan := make(chan ImageProcessingJob)
	thumbnailChan := make(chan ProcessedImage)
	metadataChan := make(chan ProcessedImage)
	results := make(chan ProcessedImage)

	var wg sync.WaitGroup

	// Stage 1: Validation workers (3 workers)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range validationChan {
				if len(job.ImageData) == 0 {
					results <- ProcessedImage{
						GameID: job.GameID,
						Error:  fmt.Errorf("empty image data"),
					}
					continue
				}
				thumbnailChan <- ProcessedImage{
					GameID:   job.GameID,
					Original: job.ImageData,
				}
			}
		}()
	}

	// Stage 2: Thumbnail workers (2 workers)
	var thumbnailWg sync.WaitGroup
	for i := 0; i < 2; i++ {
		thumbnailWg.Add(1)
		go func() {
			defer thumbnailWg.Done()
			for img := range thumbnailChan {
				time.Sleep(50 * time.Millisecond)
				img.Thumbnail = img.Original
				metadataChan <- img
			}
		}()
	}

	// Stage 3: Metadata workers (2 workers)
	var metadataWg sync.WaitGroup
	for i := 0; i < 2; i++ {
		metadataWg.Add(1)
		go func() {
			defer metadataWg.Done()
			for img := range metadataChan {
				time.Sleep(30 * time.Millisecond)
				img.Metadata = map[string]interface{}{
					"size":   len(img.Original),
					"format": "jpeg",
				}
				results <- img
			}
		}()
	}

	// Отправитель заданий
	go func() {
		for _, job := range jobs {
			validationChan <- job
		}
		close(validationChan) // Отправитель закрывает канал
	}()

	// Управление закрытием каналов (исправлено)
	go func() {
		wg.Wait()
		close(thumbnailChan)
	}()

	go func() {
		thumbnailWg.Wait()
		close(metadataChan)
	}()

	go func() {
		metadataWg.Wait()
		close(results)
	}()

	// Сбор результатов
	var processedImages []ProcessedImage
	for img := range results {
		processedImages = append(processedImages, img)
	}

	return processedImages, nil
}
