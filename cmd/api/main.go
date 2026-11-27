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
	"bboard/internal/usecase/admin"
	"bboard/internal/usecase/auth"
	"bboard/internal/usecase/bid"
	"bboard/internal/usecase/master"
	"bboard/internal/usecase/review"
	"bboard/internal/usecase/task"
	"bboard/pkg/logger"
	"bboard/pkg/storage"
	"bboard/pkg/token"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Загружаем конфигурацию (.env и переменные окружения)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Инициализируем структурированный логгер
	log := logger.New(cfg.LogLevel, cfg.IsProd)
	log.WithField("env", cfg.AppEnv).Info("starting bboard API server")

	// 3. Подключаемся к PostgreSQL через пул соединений pgxpool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.WithError(err).Fatal("failed to create PostgreSQL pool")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.WithError(err).Fatal("failed to ping PostgreSQL")
	}
	log.Info("connected to PostgreSQL")

	// 4. Инициализируем инфраструктурные утилиты
	jwtManager, err := token.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize JWT manager")
	}

	photoStorage, err := storage.NewLocalPhotoStorage(cfg.UploadDir)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize local photo storage")
	}

	// 5. Собираем слой репозиториев (Infrastructure)
	userRepo := postgres.NewUserRepo(dbPool)
	refreshTokenRepo := postgres.NewRefreshTokenRepo(dbPool)
	taskRepo := postgres.NewTaskRepo(dbPool)
	masterRepo := postgres.NewMasterRepo(dbPool)
	bidRepo := postgres.NewBidRepo(dbPool)
	reviewRepo := postgres.NewReviewRepo(dbPool)

	// Bootstrap первого администратора по ADMIN_EMAIL (идемпотентно)
	if cfg.AdminEmail != "" {
		if err := userRepo.PromoteToAdminByEmail(ctx, cfg.AdminEmail); err != nil {
			log.WithError(err).WithField("admin_email", cfg.AdminEmail).Warn("failed to bootstrap admin role by email")
		} else {
			log.WithField("admin_email", cfg.AdminEmail).Info("admin bootstrap check completed")
		}
	}

	// 6. Собираем слой бизнес-сценариев (UseCases)
	registerUC := auth.NewRegisterUseCase(userRepo, refreshTokenRepo, jwtManager, cfg.RefreshTokenTTL, cfg.AdminEmail, log)
	loginUC := auth.NewLoginUseCase(userRepo, refreshTokenRepo, jwtManager, cfg.RefreshTokenTTL, log)
	refreshUC := auth.NewRefreshUseCase(userRepo, refreshTokenRepo, jwtManager, cfg.RefreshTokenTTL, log)
	logoutUC := auth.NewLogoutUseCase(refreshTokenRepo, log)

	createTaskUC := task.NewCreateTaskUseCase(taskRepo, userRepo, log)
	listTasksUC := task.NewListTasksUseCase(taskRepo, log)
	updateTaskUC := task.NewUpdateTaskUseCase(taskRepo, log)
	deleteTaskUC := task.NewDeleteTaskUseCase(taskRepo, photoStorage, log)
	assignMasterUC := task.NewAssignMasterUseCase(taskRepo, masterRepo, log)
	markCompletedUC := task.NewMarkTaskCompletedUseCase(taskRepo, masterRepo, log)
	completeTaskUC := task.NewCompleteTaskUseCase(taskRepo, log)
	cancelTaskUC := task.NewCancelTaskUseCase(taskRepo, log)
	uploadPhotosUC := task.NewUploadTaskPhotosUseCase(taskRepo, photoStorage, log)
	deletePhotoUC := task.NewDeleteTaskPhotoUseCase(taskRepo, photoStorage, log)

	upsertProfileUC := master.NewUpsertProfileUseCase(masterRepo, userRepo, log)
	getMyProfileUC := master.NewGetMyProfileUseCase(masterRepo, log)
	getPublicProfileUC := master.NewGetPublicProfileUseCase(masterRepo, reviewRepo, log)
	deleteMyProfileUC := master.NewDeleteMyProfileUseCase(masterRepo, log)
	createServiceUC := master.NewCreateServiceUseCase(masterRepo, log)
	updateServiceUC := master.NewUpdateServiceUseCase(masterRepo, log)
	deleteServiceUC := master.NewDeleteServiceUseCase(masterRepo, log)
	listServicesUC := master.NewListServicesUseCase(masterRepo, log)

	createBidUC := bid.NewCreateBidUseCase(bidRepo, taskRepo, masterRepo, log)
	listBidsUC := bid.NewListBidsUseCase(bidRepo, taskRepo, log)

	createReviewUC := review.NewCreateReviewUseCase(reviewRepo, taskRepo, log)
	updateReviewUC := review.NewUpdateReviewUseCase(reviewRepo, log)
	deleteReviewUC := review.NewDeleteReviewUseCase(reviewRepo, log)

	listUsersUC := admin.NewListUsersUseCase(userRepo, log)
	updateUserRoleUC := admin.NewUpdateUserRoleUseCase(userRepo, log)

	// 7. Собираем слой доставки (Delivery / HTTP)
	authHandler := deliveryHTTP.NewAuthHandler(registerUC, loginUC, refreshUC, logoutUC, userRepo, cfg.RefreshTokenTTL, cfg.IsProd, log)
	taskHandler := deliveryHTTP.NewTaskHandler(createTaskUC, listTasksUC, updateTaskUC, deleteTaskUC, assignMasterUC, markCompletedUC, completeTaskUC, cancelTaskUC, uploadPhotosUC, deletePhotoUC, taskRepo, masterRepo, reviewRepo, log)
	masterHandler := deliveryHTTP.NewMasterHandler(upsertProfileUC, getMyProfileUC, getPublicProfileUC, deleteMyProfileUC, createServiceUC, updateServiceUC, deleteServiceUC, listServicesUC, log)
	bidHandler := deliveryHTTP.NewBidHandler(createBidUC, listBidsUC, log)
	reviewHandler := deliveryHTTP.NewReviewHandler(createReviewUC, updateReviewUC, deleteReviewUC, log)
	adminHandler := deliveryHTTP.NewAdminHandler(listUsersUC, updateUserRoleUC, log)
	router := deliveryHTTP.NewRouter(authHandler, taskHandler, masterHandler, bidHandler, reviewHandler, adminHandler, jwtManager, userRepo, cfg.UploadDir, log)

	// 8. Настраиваем HTTP-сервер с таймаутами
	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
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
