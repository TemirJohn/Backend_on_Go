package concurrent

import (
	"awesomeProject/db"
	"awesomeProject/models"
	"context"
	"fmt"
	"sync"
	"time"
)

/*
1. FetchGameWithDetails - Параллельная загрузка связанных данных игры
   WHY: Запросы к БД независимы и могут выполняться одновременно
   BENEFIT: Время ответа 300ms → 100ms (3x быстрее)

2. ProcessBulkGames - Массовая обработка игр
   WHY: Обработка каждой игры независима от других
   BENEFIT: 1000 игр: 100s → 10s с 10 воркерами

3. SendNotifications - Отправка уведомлений множеству пользователей
   WHY: Network I/O операции можно выполнять параллельно
   BENEFIT: 1000 пользователей: 1000s → 10s с pool воркеров

4. CalculateDashboardStats - Расчет статистики
   WHY: Каждый COUNT(*) запрос независим
   BENEFIT: 4 запроса: 400ms → 100ms (параллельно)

5. SearchGames - Параллельный поиск по разным критериям
   WHY: Поиск по названию, описанию, категории можно делать параллельно
   BENEFIT: Более быстрый и гибкий поиск
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

// FetchGameWithDetails загружает детали игры параллельно
// Использует goroutines для одновременной загрузки связанных данных
func FetchGameWithDetails(gameID uint) (*GameDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := &GameDetails{}

	// Каналы для получения результатов
	gameChan := make(chan error, 1)
	reviewsChan := make(chan []models.Review, 1)
	relatedChan := make(chan []models.Game, 1)
	statsChan := make(chan GameStatistics, 1)

	var wg sync.WaitGroup
	wg.Add(4)

	// Goroutine 1: Загрузка основной информации об игре
	go func() {
		defer wg.Done()
		var game models.Game
		err := db.DB.Preload("Category").First(&game, gameID).Error
		if err != nil {
			gameChan <- err
			return
		}
		result.Game = game
		gameChan <- nil
	}()

	// Goroutine 2: Загрузка отзывов
	go func() {
		defer wg.Done()
		var reviews []models.Review
		err := db.DB.Where("game_id = ?", gameID).
			Preload("User").
			Order("created_at DESC").
			Limit(10).
			Find(&reviews).Error

		if err != nil {
			reviewsChan <- nil
		} else {
			reviewsChan <- reviews
		}
	}()

	// Goroutine 3: Загрузка похожих игр (после получения категории)
	go func() {
		defer wg.Done()
		// Ждем загрузки основной игры
		select {
		case <-gameChan:
		case <-ctx.Done():
			relatedChan <- nil
			return
		}

		var related []models.Game
		if result.Game.ID != 0 {
			db.DB.Where("category_id = ? AND id != ?", result.Game.CategoryID, gameID).
				Limit(5).
				Find(&related)
		}
		relatedChan <- related
	}()

	// Goroutine 4: Расчет статистики
	go func() {
		defer wg.Done()
		stats := GameStatistics{}

		// Подзапросы в параллель
		var statsWg sync.WaitGroup
		statsWg.Add(4)

		// Количество отзывов
		go func() {
			defer statsWg.Done()
			db.DB.Model(&models.Review{}).Where("game_id = ?", gameID).Count(&stats.TotalReviews)
		}()

		// Средний рейтинг
		go func() {
			defer statsWg.Done()
			var avg struct{ Avg float64 }
			db.DB.Model(&models.Review{}).
				Select("AVG(rating) as avg").
				Where("game_id = ?", gameID).
				Scan(&avg)
			stats.AverageRating = avg.Avg
		}()

		// Количество владельцев
		go func() {
			defer statsWg.Done()
			db.DB.Model(&models.Ownership{}).Where("game_id = ?", gameID).Count(&stats.TotalOwners)
		}()

		// Игры той же категории (после загрузки основной игры)
		go func() {
			defer statsWg.Done()
			select {
			case <-gameChan:
			case <-ctx.Done():
				return
			}
			if result.Game.ID != 0 {
				db.DB.Model(&models.Game{}).
					Where("category_id = ?", result.Game.CategoryID).
					Count(&stats.SameCategory)
			}
		}()

		statsWg.Wait()
		statsChan <- stats
	}()

	// Ждем завершения всех goroutines
	go func() {
		wg.Wait()
		close(gameChan)
		close(reviewsChan)
		close(relatedChan)
		close(statsChan)
	}()

	// Собираем результаты с таймаутом
	select {
	case err := <-gameChan:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout fetching game details")
	}

	result.Reviews = <-reviewsChan
	result.RelatedGames = <-relatedChan
	result.Statistics = <-statsChan

	return result, nil
}

// ==================== 2. BULK OPERATIONS WITH WORKER POOL ====================

type GameProcessingJob struct {
	Game   models.Game
	Action string // "validate", "update_prices", "check_inventory", etc.
}

type GameProcessingResult struct {
	GameID  uint
	Success bool
	Error   error
	Message string
}

// ProcessBulkGames обрабатывает множество игр параллельно
// Использует worker pool pattern для контролируемого параллелизма
func ProcessBulkGames(games []models.Game, action string, numWorkers int) ([]GameProcessingResult, error) {
	if numWorkers <= 0 {
		numWorkers = 10
	}

	jobs := make(chan GameProcessingJob, len(games))
	results := make(chan GameProcessingResult, len(games))

	// Запускаем воркеров
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

	// Отправляем задания
	go func() {
		for _, game := range games {
			jobs <- GameProcessingJob{
				Game:   game,
				Action: action,
			}
		}
		close(jobs)
	}()

	// Ждем завершения всех воркеров
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
	// Симуляция обработки
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
		// Обновление цены с дисконтом
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
	Type    string // "email", "push", "sms"
}

type NotificationResult struct {
	UserID  uint
	Success bool
	Error   error
}

// SendBulkNotifications отправляет уведомления параллельно
// Использует ограниченное количество воркеров для защиты от перегрузки
func SendBulkNotifications(notifications []NotificationJob, maxWorkers int) ([]NotificationResult, error) {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	results := make(chan NotificationResult, len(notifications))

	// Semaphore для ограничения параллельности
	semaphore := make(chan struct{}, maxWorkers)

	var wg sync.WaitGroup

	// Запуск воркеров
	for _, job := range notifications {
		wg.Add(1)
		go func(j NotificationJob) {
			defer wg.Done()

			// Ждем свободного слота
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := sendNotification(j)
			results <- result
		}(job)
	}

	// Ждем завершения
	go func() {
		wg.Wait()
		close(results)
	}()

	// Собираем результаты
	var allResults []NotificationResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults, nil
}

func sendNotification(job NotificationJob) NotificationResult {
	// Симуляция отправки уведомления
	time.Sleep(100 * time.Millisecond)

	// Проверка пользователя
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

	// Симуляция успешной отправки
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

// CalculateDashboardStats вычисляет статистику параллельно
// Каждый COUNT(*) запрос выполняется в отдельной goroutine
func CalculateDashboardStats() (*DashboardStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats := &DashboardStats{}
	var wg sync.WaitGroup

	// Канал для ошибок
	errChan := make(chan error, 8)

	// 1. Общее количество пользователей
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := db.DB.Model(&models.User{}).Count(&stats.TotalUsers).Error; err != nil {
			errChan <- fmt.Errorf("users count: %w", err)
		}
	}()

	// 2. Общее количество игр
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := db.DB.Model(&models.Game{}).Count(&stats.TotalGames).Error; err != nil {
			errChan <- fmt.Errorf("games count: %w", err)
		}
	}()

	// 3. Общее количество отзывов
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := db.DB.Model(&models.Review{}).Count(&stats.TotalReviews).Error; err != nil {
			errChan <- fmt.Errorf("reviews count: %w", err)
		}
	}()

	// 4. Общее количество продаж
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := db.DB.Model(&models.Ownership{}).Count(&stats.TotalSales).Error; err != nil {
			errChan <- fmt.Errorf("sales count: %w", err)
		}
	}()

	// 5. Активные пользователи (не забаненные)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := db.DB.Model(&models.User{}).
			Where("is_banned = ?", false).
			Count(&stats.ActiveUsers).Error; err != nil {
			errChan <- fmt.Errorf("active users: %w", err)
		}
	}()

	// 6. Недавние игры (за последние 30 дней)
	wg.Add(1)
	go func() {
		defer wg.Done()
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		if err := db.DB.Model(&models.Game{}).
			Where("created_at > ?", thirtyDaysAgo).
			Count(&stats.RecentGames).Error; err != nil {
			errChan <- fmt.Errorf("recent games: %w", err)
		}
	}()

	// 7. Средний рейтинг
	wg.Add(1)
	go func() {
		defer wg.Done()
		var avg struct{ Avg float64 }
		if err := db.DB.Model(&models.Review{}).
			Select("AVG(rating) as avg").
			Scan(&avg).Error; err != nil {
			errChan <- fmt.Errorf("average rating: %w", err)
		} else {
			stats.AverageRating = avg.Avg
		}
	}()

	// 8. Топ категория
	wg.Add(1)
	go func() {
		defer wg.Done()
		var result struct {
			CategoryID uint
			Count      int64
		}
		err := db.DB.Model(&models.Game{}).
			Select("category_id, COUNT(*) as count").
			Group("category_id").
			Order("count DESC").
			Limit(1).
			Scan(&result).Error

		if err != nil {
			errChan <- fmt.Errorf("top category: %w", err)
			return
		}

		var category models.Category
		if err := db.DB.First(&category, result.CategoryID).Error; err == nil {
			stats.TopCategory = category.Name
		}
	}()

	// Ждем завершения с таймаутом
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errChan)
	}()

	select {
	case <-done:
		// Проверяем ошибки
		for err := range errChan {
			if err != nil {
				stats.Error = err
				return stats, err
			}
		}
		return stats, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout calculating stats")
	}
}

// ==================== 5. PARALLEL SEARCH ====================

type SearchResult struct {
	Games      []models.Game
	TotalFound int
	SearchTime time.Duration
}

// ParallelSearch выполняет поиск по разным критериям параллельно
func ParallelSearch(query string) (*SearchResult, error) {
	start := time.Now()

	// Каналы для результатов
	nameResults := make(chan []models.Game, 1)
	descResults := make(chan []models.Game, 1)
	categoryResults := make(chan []models.Game, 1)

	var wg sync.WaitGroup
	wg.Add(3)

	// Поиск по названию
	go func() {
		defer wg.Done()
		var games []models.Game
		db.DB.Where("name ILIKE ?", "%"+query+"%").
			Limit(20).
			Find(&games)
		nameResults <- games
	}()

	// Поиск по описанию
	go func() {
		defer wg.Done()
		var games []models.Game
		db.DB.Where("description ILIKE ?", "%"+query+"%").
			Limit(20).
			Find(&games)
		descResults <- games
	}()

	// Поиск по категории
	go func() {
		defer wg.Done()
		var category models.Category
		err := db.DB.Where("name ILIKE ?", "%"+query+"%").First(&category).Error
		if err != nil {
			categoryResults <- nil
			return
		}

		var games []models.Game
		db.DB.Where("category_id = ?", category.ID).
			Limit(20).
			Find(&games)
		categoryResults <- games
	}()

	// Ждем результатов
	wg.Wait()

	// Объединяем результаты (удаляем дубликаты)
	resultMap := make(map[uint]models.Game)

	for _, game := range <-nameResults {
		resultMap[game.ID] = game
	}
	for _, game := range <-descResults {
		resultMap[game.ID] = game
	}
	if catGames := <-categoryResults; catGames != nil {
		for _, game := range catGames {
			resultMap[game.ID] = game
		}
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

// ProcessImagesInPipeline обрабатывает изображения через pipeline
func ProcessImagesInPipeline(jobs []ImageProcessingJob) ([]ProcessedImage, error) {
	// Stage 1: Validation
	validationChan := make(chan ImageProcessingJob)

	// Stage 2: Thumbnail generation
	thumbnailChan := make(chan ProcessedImage)

	// Stage 3: Metadata extraction
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
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for img := range thumbnailChan {
				// Симуляция генерации thumbnail
				time.Sleep(50 * time.Millisecond)
				img.Thumbnail = img.Original // В реальности - resize
				metadataChan <- img
			}
		}()
	}

	// Stage 3: Metadata workers (2 workers)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for img := range metadataChan {
				// Симуляция извлечения metadata
				time.Sleep(30 * time.Millisecond)
				img.Metadata = map[string]interface{}{
					"size":   len(img.Original),
					"format": "jpeg",
				}
				results <- img
			}
		}()
	}

	// Отправка заданий
	go func() {
		for _, job := range jobs {
			validationChan <- job
		}
		close(validationChan)
	}()

	// Закрытие каналов
	go func() {
		wg.Wait()
		close(thumbnailChan)
		close(metadataChan)
		close(results)
	}()

	// Сбор результатов
	var processedImages []ProcessedImage
	for img := range results {
		processedImages = append(processedImages, img)
	}

	return processedImages, nil
}
