package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	"go.etcd.io/etcd/raft/v3/raftpb"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/algo/borda"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/algo/raft"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb/database"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb/repository/pg"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/leader"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/internal/config"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/internal/rest"
	rla "github.com/mahmud2011/swarmchestrate/resource-lead-agent/resource-lead-agent"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/server"
)

func NewCMDRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource-lead-agent",
		Short: "Responsible for managing worker clusters",
		Long: `This component will reside in raft cluster.
	It manages the worker clusters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()

			spew.Dump("configuratons:", cfg)

			r := chi.NewRouter()
			r.Use(middleware.Logger)
			r.Use(middleware.Recoverer)
			r.Use(middleware.Heartbeat("/health"))

			postgresDB := database.NewPostgresDB(cfg)
			clusterRepo := pg.NewClusterRepository(postgresDB.DB)
			nodeRepo := pg.NewNodeRepository(postgresDB.DB)
			applicationRepo := pg.NewApplicationRepository(postgresDB.DB)

			proposeC := make(chan string)
			defer close(proposeC)
			confChangeC := make(chan raftpb.ConfChange)
			defer close(confChangeC)
			rn, snapshotterReady, errorC := raft.NewRaftNode(cfg.Raft.ID, cfg.Raft.Peers, false, func() ([]byte, error) { return []byte{}, nil }, proposeC, confChangeC)

			<-snapshotterReady

			kbSvc := kb.NewService(clusterRepo, nodeRepo, applicationRepo)
			leaderSvc := leader.NewService(cfg)
			rlaSvc := rla.NewService(kbSvc, rn, leaderSvc, &borda.Borda{})

			go func() {
				for {
					time.Sleep(5 * time.Second)
				}
			}()

			for {
				time.Sleep(5 * time.Second)

				spew.Dump("registration attempt ...")

				if err := rlaSvc.RegisterCluster(context.Background(), cfg); err == nil {
					break
				} else {
					spew.Dump(err)
				}
			}

			apiRouter := chi.NewRouter()
			r.Mount("/swarmchestrate/api", apiRouter)

			rest.NewRLAHandler(apiRouter, rlaSvc)

			srv := server.NewHttpServer(*cfg.HTTPServer, r)
			wait := make(chan os.Signal, 1)
			signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatalln(err)
				}
			}()

			slog.Info("server started at ...........")

			ticker := time.NewTicker(cfg.App.SchedulerInterval)
			defer ticker.Stop()

			go func() {
				var wg sync.WaitGroup

				for {
					select {
					case <-ticker.C:
						wg.Add(1)
						go func() {
							defer wg.Done()
							if err := rlaSvc.SchedulePendingApps(); err != nil {
								log.Printf("Error scheduling apps: %v", err)
							}
						}()

						wg.Add(1)
						go func() {
							defer wg.Done()
							if err := rlaSvc.HandleStalledApps(); err != nil {
								log.Printf("Error handling stalled apps: %v", err)
							}
						}()
						wg.Wait()
					case <-wait:
						return
					}
				}
			}()

			select {
			case err := <-errorC:
				fmt.Printf("Error from raft node: %v\n", err)
			case sig := <-wait:
				fmt.Printf("received OS signal: %v\n", sig)
			}

			ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPServer.GracefulShutdownTimeout*time.Second)
			defer cancel()

			slog.Info("server shutting down ...........")
			if err := srv.Stop(ctx); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
