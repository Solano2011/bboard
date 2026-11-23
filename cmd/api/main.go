package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bboard/internal/config"
	deliveryHTTP "bboard/internal/delivery/http"
	"bboard/internal/repository/postgres"
	"bboard/internal/usecase/auth"
	"bboard/internal/usecase/bid"
	"bboard/internal/usecase/master"
	"bboard/internal/usecase/task"
	"bboard/pkg/logger"
	"bboard/pkg/token"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Инициализируем структурированный логгер logrus
	log := logger.New(cfg.LogLevel, cfg.IsProd)
	log.WithField("env", cfg.AppEnv).Info("starting bboard API server")

	// 3. Подключаемся к PostgreSQL с проверкой соединения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.WithError(err).Fatal("failed to create postgres connection pool")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.WithError(err).Fatal("failed to ping postgres database")
	}
	log.Info("connected to PostgreSQL")

	// 4. Инициализируем инфраструктурные утилиты
	jwtManager, err := token.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize JWT manager")
	}

	// 5. Собираем слой репозиториев (Infrastructure)
	userRepo := postgres.NewUserRepo(dbPool)
	taskRepo := postgres.NewTaskRepo(dbPool)
	masterRepo := postgres.NewMasterRepo(dbPool)
	bidRepo := postgres.NewBidRepo(dbPool)

	// 6. Собираем слой бизнес-сценариев (UseCases)
	registerUC := auth.NewRegisterUseCase(userRepo, log)
	loginUC := auth.NewLoginUseCase(userRepo, jwtManager, log)

	createTaskUC := task.NewCreateTaskUseCase(taskRepo, userRepo, log)
	listTasksUC := task.NewListTasksUseCase(taskRepo, log)
	assignMasterUC := task.NewAssignMasterUseCase(taskRepo, masterRepo, log)
	completeTaskUC := task.NewCompleteTaskUseCase(taskRepo, log)
	cancelTaskUC := task.NewCancelTaskUseCase(taskRepo, log)

	createMasterUC := master.NewCreateProfileUseCase(masterRepo, userRepo, log)
	listMastersUC := master.NewListProfilesUseCase(masterRepo, log)

	createBidUC := bid.NewCreateBidUseCase(bidRepo, taskRepo, masterRepo, log)
	listBidsUC := bid.NewListBidsUseCase(bidRepo, taskRepo, log)

	// 7. Собираем слой доставки (Delivery / HTTP)
	authHandler := deliveryHTTP.NewAuthHandler(registerUC, loginUC, log)
	taskHandler := deliveryHTTP.NewTaskHandler(createTaskUC, listTasksUC, assignMasterUC, completeTaskUC, cancelTaskUC, log)
	masterHandler := deliveryHTTP.NewMasterHandler(createMasterUC, listMastersUC, log)
	bidHandler := deliveryHTTP.NewBidHandler(createBidUC, listBidsUC, log)
	router := deliveryHTTP.NewRouter(authHandler, taskHandler, masterHandler, bidHandler, jwtManager, log)

	// 8. Настраиваем HTTP-сервер с таймаутами
	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		log.WithField("port", cfg.HTTPPort).Info("HTTP server is listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("HTTP server failed unexpectedly")
		}
	}()

	// 9. Ожидаем сигнал завершения (Ctrl+C / SIGTERM) для Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down HTTP server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("forced server shutdown")
	}

	log.Info("server stopped cleanly")
}
