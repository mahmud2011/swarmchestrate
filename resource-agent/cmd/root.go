package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"

	rl "github.com/mahmud2011/swarmchestrate/resource-agent/external/resource-lead"
	"github.com/mahmud2011/swarmchestrate/resource-agent/internal/config"
	ra "github.com/mahmud2011/swarmchestrate/resource-agent/resource-agent"
	"github.com/mahmud2011/swarmchestrate/resource-agent/server"
)

func NewCMDRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource-agent",
		Short: "Tracks resource and application status",
		Long: `This component resides in each cluster.
It tracks resource status and running application status and informs resource-lead(s).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()

			spew.Dump("configuratons:", cfg)

			r := chi.NewRouter()
			r.Use(middleware.Logger)
			r.Use(middleware.Recoverer)
			r.Use(middleware.Heartbeat("/health"))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("."))
			})

			resourceLeadSvc := rl.NewService(cfg)
			resourceAgentSvc := ra.NewService(resourceLeadSvc)
			srv := server.NewHttpServer(*cfg.HTTPServer, r)

			// Wait for a successful cluster registration.
			for {
				time.Sleep(5 * time.Second)
				if err := resourceAgentSvc.RegisterCluster(context.Background(), cfg); err == nil {
					break
				} else {
					slog.Error("RegisterCluster failed", "error", err)
				}
			}

			// Start the HTTP server.
			go func() {
				if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error("HTTP server error", "error", err)
					os.Exit(1)
				}
			}()
			slog.Info("server started", "port", cfg.HTTPServer.Port)

			// Prepare signal handling.
			wait := make(chan os.Signal, 1)
			signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)

			ticker := time.NewTicker(cfg.App.HeartBeatInterval)
			defer ticker.Stop()

			// Helper to run tasks concurrently.
			runTask := func(wg *sync.WaitGroup, taskName string, task func() error) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := task(); err != nil {
						slog.Error(taskName, "error", err)
					}
				}()
			}

			// Launch periodic tasks.
			go func() {
				var wg sync.WaitGroup
				for {
					select {
					case <-ticker.C:
						runTask(&wg, "SyncClusterConfig", func() error {
							return resourceAgentSvc.SyncClusterConfig(context.Background())
						})
						runTask(&wg, "SendNodeInfo", func() error {
							nodes, err := resourceAgentSvc.CollectNodeInfo()
							if err != nil {
								return err
							}
							return resourceAgentSvc.SendNodeInfo(context.Background(), nodes)
						})
						runTask(&wg, "DeployScheduledJobs", func() error {
							apps, err := resourceAgentSvc.GetScheduledJobs(context.Background())
							if err != nil {
								return err
							}
							return resourceAgentSvc.DeployScheduledJobs(apps)
						})
						runTask(&wg, "SendApplicationStatus", func() error {
							return resourceAgentSvc.SendApplicationStatus()
						})
						wg.Wait()
					case <-wait:
						return
					}
				}
			}()

			<-wait

			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				cfg.HTTPServer.GracefulShutdownTimeout*time.Second,
			)
			defer cancel()

			slog.Info("server shutting down")
			if err := srv.Stop(shutdownCtx); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
