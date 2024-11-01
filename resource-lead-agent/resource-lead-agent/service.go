package resourceleadagent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain/dto"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/internal/config"
)

type IKBService interface {
	CreateCluster(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error)
	GetCluster(ctx context.Context, ctr *domain.ClusterFetchCtr) (*domain.Cluster, error)
	ListClusters(ctx context.Context, ctr *domain.ClusterListCtr) ([]*domain.Cluster, error)
	UpdateCluster(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error)
	DeleteCluster(ctx context.Context, m *domain.Cluster) error

	UpsertNodes(ctx context.Context, nodes []*domain.Node) ([]*domain.Node, error)
	ListNodes(ctx context.Context, ctr *domain.NodeListCtr) ([]*domain.Node, error)
	DeleteNode(ctx context.Context, m *domain.Node) error

	CreateApplication(ctx context.Context, m *domain.Application) (*domain.Application, error)
	GetApplication(ctx context.Context, ctr *domain.ApplicationFetchCtr) (*domain.Application, error)
	ListApplications(ctx context.Context, ctr *domain.ApplicationListCtr) ([]*domain.Application, error)
	UpdateApplication(ctx context.Context, m *domain.Application) (*domain.Application, error)
	DeleteApplication(ctx context.Context, m *domain.Application) error
}

type ILeaderService interface {
	CreateCluster(ctx context.Context, leaderHost string, d *dto.AddClusterReq) (*domain.Cluster, error)
	UpdateCluster(ctx context.Context, leaderHost string, d *dto.UpdateClusterReq) (*domain.Cluster, error)
}

type IRaftService interface {
	IsLeader() bool
	GetLeaderID() uint64
}

type INodeScorer interface {
	ScoreAndFilterNodes(nodes []*domain.Node, qos domain.QoS) (bestClusterID string, filteredNodes []string)
}

type Service struct {
	raftSvc   IRaftService
	kbSvc     IKBService
	leaderSvc ILeaderService
	scorerSvc INodeScorer
}

func NewService(kbs IKBService, rs IRaftService, ls ILeaderService, nss INodeScorer) *Service {
	return &Service{
		raftSvc:   rs,
		kbSvc:     kbs,
		leaderSvc: ls,
		scorerSvc: nss,
	}
}

func (s *Service) Isleader() bool {
	return s.raftSvc.IsLeader()
}

func (s *Service) GetLeaderHost(ctx context.Context) (string, error) {
	leaderRaftID := s.raftSvc.GetLeaderID()
	if leaderRaftID == 0 {
		return "", errors.New("still waiting for leader information")
	}

	leaderRaftIDInt := int(leaderRaftID)

	leaderCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{RaftID: &leaderRaftIDInt})
	if err != nil {
		return "", err
	}

	return leaderCluster.IP, nil
}

func (s *Service) RegisterCluster(ctx context.Context, cfg *config.Config) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Error building in-cluster config: %v", err)
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Error creating Kubernetes client: %v", err)
		return err
	}

	cm, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Get(context.TODO(), "self", metav1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			iSvc, err := clientset.CoreV1().Services("ingress-nginx").Get(context.Background(), "ingress-nginx-controller", metav1.GetOptions{})
			if err != nil {
				return err
			}

			if len(iSvc.Status.LoadBalancer.Ingress) == 0 {
				return err
			}

			var ip string
			for _, lb := range iSvc.Status.LoadBalancer.Ingress {
				if lb.IP != "" {
					ip = lb.IP
					break
				}
			}

			req := &dto.AddClusterReq{
				Type:   domain.ClusterType(cfg.App.Type),
				Role:   domain.ClusterRoleMaster,
				IP:     ip,
				RaftID: &cfg.Raft.ID,
			}

			createdCluster, err := s.CreateCluster(ctx, req)
			if err != nil {
				return err
			}

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "self",
				},
				Data: map[string]string{
					"id":   createdCluster.ID,
					"role": string(createdCluster.Role),
				},
			}

			if _, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Create(context.TODO(), cm, metav1.CreateOptions{}); err != nil {
				return err
			}

			return nil
		}

		return err
	}

	if cm.Data["role"] == string(domain.ClusterRoleWorker) {
		req := &dto.UpdateClusterReq{
			ID:     cm.Data["id"],
			Role:   domain.ClusterRoleMaster,
			RaftID: &cfg.Raft.ID,
		}
		updatedCluster, err := s.UpdateCluster(ctx, req)
		if err != nil {
			return err
		}

		cm.Data["role"] = string(updatedCluster.Role)

		if _, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Update(context.TODO(), cm, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) CreateCluster(ctx context.Context, d *dto.AddClusterReq) (*domain.Cluster, error) {
	if !s.raftSvc.IsLeader() {
		leaderRaftID := s.raftSvc.GetLeaderID()
		if leaderRaftID == 0 {
			return nil, errors.New("still waiting for leader information")
		}

		leaderRaftIDInt := int(leaderRaftID)

		leaderCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{RaftID: &leaderRaftIDInt})
		if err != nil {
			spew.Dump(err)
			return nil, err
		}

		createdCluster, err := s.leaderSvc.CreateCluster(ctx, leaderCluster.IP, d)
		if err != nil {
			spew.Dump(err)
			return nil, err
		}

		return createdCluster, nil
	}

	m := &domain.Cluster{
		Type: d.Type,
		Role: d.Role,
		IP:   d.IP,
	}

	if d.RaftID != nil {
		m.RaftID = d.RaftID
	}

	cluster, err := s.kbSvc.CreateCluster(ctx, m)
	if err != nil {
		spew.Dump(err)
		log.Println("failed to create cluster in kb", err)

		return nil, err
	}

	return cluster, nil
}

func (s *Service) GetCluster(ctx context.Context, ctr *domain.ClusterFetchCtr) (*domain.Cluster, error) {
	cluster, err := s.kbSvc.GetCluster(ctx, ctr)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (s *Service) ListClusters(ctx context.Context, ctr *domain.ClusterListCtr) ([]*domain.Cluster, error) {
	clusters, err := s.kbSvc.ListClusters(ctx, ctr)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (s *Service) UpdateCluster(ctx context.Context, d *dto.UpdateClusterReq) (*domain.Cluster, error) {
	if !s.raftSvc.IsLeader() {
		leaderRaftID := s.raftSvc.GetLeaderID()
		leaderRaftIDInt := int(leaderRaftID)

		leaderCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{RaftID: &leaderRaftIDInt})
		if err != nil {
			return nil, err
		}

		createdCluster, err := s.leaderSvc.UpdateCluster(ctx, leaderCluster.IP, d)
		if err != nil {
			return nil, err
		}

		return createdCluster, nil
	}

	existingCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
		ClusterID: &d.ID,
	})
	if err != nil {
		return nil, err
	}

	if existingCluster == nil {
		return nil, nil
	}

	if d.Type != "" {
		existingCluster.Type = d.Type
	}

	if d.Role != "" {
		existingCluster.Role = d.Role
	}

	if d.IP != "" {
		existingCluster.IP = d.IP
	}

	if d.RaftID != nil {
		existingCluster.RaftID = d.RaftID
	}

	updatedCluster, err := s.kbSvc.UpdateCluster(ctx, existingCluster)
	if err != nil {
		return nil, err
	}

	return updatedCluster, nil
}

func (s *Service) DeleteCluster(ctx context.Context, id string) error {
	m := &domain.Cluster{
		ID: id,
	}

	if err := s.kbSvc.DeleteCluster(ctx, m); err != nil {
		return err
	}

	return nil
}

func (s *Service) SyncClusterConfig(ctx context.Context) (map[string]string, error) {
	clusters, err := s.kbSvc.ListClusters(ctx, &domain.ClusterListCtr{})
	if err != nil {
		return nil, err
	}

	masters := []string{}

	resp := map[string]string{}

	for _, cluster := range clusters {
		resp[cluster.ID] = fmt.Sprintf(`{"ip": "%s"}`, cluster.IP)

		if cluster.Role == domain.ClusterRoleMaster {
			masters = append(masters, cluster.ID)
		}
	}

	resp["masters"] = strings.Join(masters, ",")

	return resp, nil
}

func (s *Service) UpsertNodes(ctx context.Context, clusterID string, un *dto.UpsertNode) ([]*domain.Node, error) {
	for i := range un.Nodes {
		un.Nodes[i].ClusterID = un.ClusterID
	}

	upsertedNodes, err := s.kbSvc.UpsertNodes(ctx, un.Nodes)
	if err != nil {
		return nil, err
	}

	return upsertedNodes, nil
}

func (s *Service) CreateApplication(ctx context.Context, d *dto.AddApplicationReq) (*domain.Application, error) {
	m := &domain.Application{
		Name:     d.Name,
		Labels:   d.Labels,
		QoS:      d.QoS,
		CloudSvc: d.CloudSvc,
		FogSvc:   d.FogSvc,
		EdgeSvc:  d.EdgeSvc,
	}

	cloudAppVersion, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	fogAppVersion, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	edgeAppVersion, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	if m.CloudSvc != nil {
		version := cloudAppVersion.String()
		m.CloudSvcVersion = &version
	}

	if m.FogSvc != nil {
		version := fogAppVersion.String()
		m.FogSvcVersion = &version
	}

	if m.EdgeSvc != nil {
		version := edgeAppVersion.String()
		m.EdgeSvcVersion = &version
	}

	application, err := s.kbSvc.CreateApplication(ctx, m)
	if err != nil {
		return nil, err
	}

	return application, nil
}

func (s *Service) GetApplication(ctx context.Context, ctr *domain.ApplicationFetchCtr) (*domain.Application, error) {
	application, err := s.kbSvc.GetApplication(ctx, ctr)
	if err != nil {
		return nil, err
	}
	return application, nil
}

func (s *Service) ListApplications(ctx context.Context, ctr *domain.ApplicationListCtr) ([]*domain.Application, error) {
	applications, err := s.kbSvc.ListApplications(ctx, ctr)
	if err != nil {
		return nil, err
	}

	return applications, nil
}

func (s *Service) UpdateApplication(ctx context.Context, d *dto.UpdateApplicationReq) (*domain.Application, error) {
	existingApplication, err := s.kbSvc.GetApplication(ctx, &domain.ApplicationFetchCtr{
		ApplicationID: &d.ID,
	})
	if err != nil {
		return nil, err
	}

	if existingApplication == nil {
		return nil, nil
	}

	cloudAppVersion, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	cloudAppVersionStr := cloudAppVersion.String()

	fogAppVersion, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	fogAppVersionStr := fogAppVersion.String()

	edgeAppVersion, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	edgeAppVersionStr := edgeAppVersion.String()

	if d.QoS != nil && existingApplication.QoS != d.QoS {
		existingApplication.QoS = d.QoS

		if existingApplication.CloudSvc != nil {
			existingApplication.CloudSvcCluster = nil
			existingApplication.CloudSvcNodes = nil
			existingApplication.CloudSvcStatus = nil
			existingApplication.CloudSvcHeartbeat = nil
			existingApplication.CloudSvcVersion = &cloudAppVersionStr
		}

		if existingApplication.FogSvc != nil {
			existingApplication.FogSvcCluster = nil
			existingApplication.FogSvcNodes = nil
			existingApplication.FogSvcStatus = nil
			existingApplication.FogSvcHeartbeat = nil
			existingApplication.FogSvcVersion = &fogAppVersionStr
		}

		if existingApplication.EdgeSvc != nil {
			existingApplication.EdgeSvcCluster = nil
			existingApplication.EdgeSvcNodes = nil
			existingApplication.EdgeSvcStatus = nil
			existingApplication.EdgeSvcHeartbeat = nil
			existingApplication.EdgeSvcVersion = &edgeAppVersionStr
		}
	}

	if d.Name != nil {
		existingApplication.Name = *d.Name

		if existingApplication.CloudSvc != nil {
			existingApplication.CloudSvcStatus = nil
			existingApplication.CloudSvcHeartbeat = nil
			existingApplication.CloudSvcVersion = &cloudAppVersionStr
		}

		if existingApplication.FogSvc != nil {
			existingApplication.FogSvcStatus = nil
			existingApplication.FogSvcHeartbeat = nil
			existingApplication.FogSvcVersion = &fogAppVersionStr
		}

		if existingApplication.EdgeSvc != nil {
			existingApplication.EdgeSvcStatus = nil
			existingApplication.EdgeSvcHeartbeat = nil
			existingApplication.EdgeSvcVersion = &edgeAppVersionStr
		}
	}

	if d.Labels != nil {
		existingApplication.Labels = d.Labels

		if existingApplication.CloudSvc != nil {
			existingApplication.CloudSvcStatus = nil
			existingApplication.CloudSvcHeartbeat = nil
			existingApplication.CloudSvcVersion = &cloudAppVersionStr
		}

		if existingApplication.FogSvc != nil {
			existingApplication.FogSvcStatus = nil
			existingApplication.FogSvcHeartbeat = nil
			existingApplication.FogSvcVersion = &fogAppVersionStr
		}

		if existingApplication.EdgeSvc != nil {
			existingApplication.EdgeSvcStatus = nil
			existingApplication.EdgeSvcHeartbeat = nil
			existingApplication.EdgeSvcVersion = &edgeAppVersionStr
		}
	}

	if d.CloudSvc != nil {
		existingApplication.CloudSvc = d.CloudSvc

		existingApplication.CloudSvcStatus = nil
		existingApplication.CloudSvcHeartbeat = nil
		existingApplication.CloudSvcVersion = &cloudAppVersionStr
	}

	if d.FogSvc != nil {
		existingApplication.FogSvc = d.FogSvc

		existingApplication.FogSvcStatus = nil
		existingApplication.FogSvcHeartbeat = nil
		existingApplication.FogSvcVersion = &fogAppVersionStr
	}

	if d.EdgeSvc != nil {
		existingApplication.EdgeSvc = d.EdgeSvc

		existingApplication.EdgeSvcStatus = nil
		existingApplication.EdgeSvcHeartbeat = nil
		existingApplication.EdgeSvcVersion = &edgeAppVersionStr
	}

	updatedApplication, err := s.kbSvc.UpdateApplication(ctx, existingApplication)
	if err != nil {
		return nil, err
	}

	return updatedApplication, nil
}

func (s *Service) DeleteApplication(ctx context.Context, id string) error {
	m := &domain.Application{
		ID: id,
	}

	if err := s.kbSvc.DeleteApplication(ctx, m); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetClusterScheduledApps(ctx context.Context, d *dto.ClusterScheduledApps) ([]*dto.ScheduledApplication, error) {
	cluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
		ClusterID: &d.ClusterID,
	})
	if err != nil {
		return nil, err
	}

	null := domain.StateNULL
	ctr := &domain.ApplicationListCtr{}
	if cluster.Type == domain.ClusterTypeCloud {
		ctr.CloudSvcCluster = &cluster.ID
		ctr.CloudSvcStatus = &null
	} else if cluster.Type == domain.ClusterTypeFog {
		ctr.FogSvcCluster = &cluster.ID
		ctr.FogSvcStatus = &null
	} else {
		ctr.EdgeSvcCluster = &cluster.ID
		ctr.EdgeSvcStatus = &null
	}

	clusterScheduledApps, err := s.kbSvc.ListApplications(ctx, ctr)
	if err != nil {
		return nil, err
	}

	var pendingDeployements []*dto.ScheduledApplication
	for _, clusterScheduledApp := range clusterScheduledApps {
		var cloudIP *string
		var fogIP *string
		var edgeIP *string

		labelReq := *clusterScheduledApp.Labels

		// Labels
		labelReq["app.kubernetes.io/part-of"] = "swarmchestrate"
		labelReq["app.kubernetes.io/component"] = "application"
		labelReq["app.kubernetes.io/name"] = clusterScheduledApp.Name
		labelReq["app.kubernetes.io/id"] = clusterScheduledApp.ID

		// IPs
		if clusterScheduledApp.CloudSvc != nil {
			cloudCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
				ClusterID: clusterScheduledApp.CloudSvcCluster,
			})
			if err != nil {
				return nil, err
			}
			cloudIP = &cloudCluster.IP
		}

		if clusterScheduledApp.FogSvc != nil {
			fogCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
				ClusterID: clusterScheduledApp.FogSvcCluster,
			})
			if err != nil {
				return nil, err
			}
			fogIP = &fogCluster.IP
		}

		if clusterScheduledApp.EdgeSvc != nil {
			edgeCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
				ClusterID: clusterScheduledApp.EdgeSvcCluster,
			})
			if err != nil {
				return nil, err
			}

			edgeIP = &edgeCluster.IP
		}

		scheduledApp := &dto.ScheduledApplication{
			CloudIP: cloudIP,
			FogIP:   fogIP,
			EdgeIP:  edgeIP,
			Application: &dto.Application{
				ID:   clusterScheduledApp.ID,
				Name: clusterScheduledApp.Name,
			},
		}

		if cluster.Type == domain.ClusterTypeCloud {
			labelReq["app.kubernetes.io/version"] = *clusterScheduledApp.CloudSvcVersion

			scheduledApp.Application.Svc = clusterScheduledApp.CloudSvc
			scheduledApp.Application.SvcVersion = clusterScheduledApp.CloudSvcVersion
			scheduledApp.Application.SvcNodes = clusterScheduledApp.CloudSvcNodes
		} else if cluster.Type == domain.ClusterTypeFog {
			labelReq["app.kubernetes.io/version"] = *clusterScheduledApp.FogSvcVersion

			scheduledApp.Application.Svc = clusterScheduledApp.FogSvc
			scheduledApp.Application.SvcVersion = clusterScheduledApp.FogSvcVersion
			scheduledApp.Application.SvcNodes = clusterScheduledApp.FogSvcNodes
		} else {
			labelReq["app.kubernetes.io/version"] = *clusterScheduledApp.EdgeSvcVersion

			scheduledApp.Application.Svc = clusterScheduledApp.EdgeSvc
			scheduledApp.Application.SvcVersion = clusterScheduledApp.EdgeSvcVersion
			scheduledApp.Application.SvcNodes = clusterScheduledApp.EdgeSvcNodes
		}

		scheduledApp.Application.Labels = &labelReq

		pendingDeployements = append(pendingDeployements, scheduledApp)
	}

	return pendingDeployements, nil
}

func (s *Service) GetClusterScheduledApp(ctx context.Context, d *dto.ClusterScheduledApp) (*dto.ScheduledApplication, error) {
	cluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
		ClusterID: &d.ClusterID,
	})
	if err != nil {
		return nil, err
	}

	ctr := &domain.ApplicationFetchCtr{
		ApplicationID: &d.AppID,
	}
	if cluster.Type == domain.ClusterTypeCloud {
		ctr.CloudSvcCluster = &cluster.ID
	} else if cluster.Type == domain.ClusterTypeFog {
		ctr.FogSvcCluster = &cluster.ID
	} else {
		ctr.EdgeSvcCluster = &cluster.ID
	}

	clusterScheduledApp, err := s.kbSvc.GetApplication(ctx, ctr)
	if err != nil {
		return nil, err
	}

	var cloudIP *string
	var fogIP *string
	var edgeIP *string

	labelReq := *clusterScheduledApp.Labels

	// Labels
	labelReq["app.kubernetes.io/part-of"] = "swarmchestrate"
	labelReq["app.kubernetes.io/component"] = "application"
	labelReq["app.kubernetes.io/name"] = clusterScheduledApp.Name
	labelReq["app.kubernetes.io/id"] = clusterScheduledApp.ID

	// IPs
	if clusterScheduledApp.CloudSvc != nil {
		cloudCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
			ClusterID: clusterScheduledApp.CloudSvcCluster,
		})
		if err != nil {
			return nil, err
		}
		cloudIP = &cloudCluster.IP
	}

	if clusterScheduledApp.FogSvc != nil {
		fogCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
			ClusterID: clusterScheduledApp.FogSvcCluster,
		})
		if err != nil {
			return nil, err
		}
		fogIP = &fogCluster.IP
	}

	if clusterScheduledApp.EdgeSvc != nil {
		edgeCluster, err := s.kbSvc.GetCluster(ctx, &domain.ClusterFetchCtr{
			ClusterID: clusterScheduledApp.EdgeSvcCluster,
		})
		if err != nil {
			return nil, err
		}

		edgeIP = &edgeCluster.IP
	}

	scheduledApp := &dto.ScheduledApplication{
		CloudIP: cloudIP,
		FogIP:   fogIP,
		EdgeIP:  edgeIP,
		Application: &dto.Application{
			ID:   clusterScheduledApp.ID,
			Name: clusterScheduledApp.Name,
		},
	}

	if cluster.Type == domain.ClusterTypeCloud {
		labelReq["app.kubernetes.io/version"] = *clusterScheduledApp.CloudSvcVersion

		scheduledApp.Application.Svc = clusterScheduledApp.CloudSvc
		scheduledApp.Application.SvcVersion = clusterScheduledApp.CloudSvcVersion
		scheduledApp.Application.SvcNodes = clusterScheduledApp.CloudSvcNodes
	} else if cluster.Type == domain.ClusterTypeFog {
		labelReq["app.kubernetes.io/version"] = *clusterScheduledApp.FogSvcVersion

		scheduledApp.Application.Svc = clusterScheduledApp.FogSvc
		scheduledApp.Application.SvcVersion = clusterScheduledApp.FogSvcVersion
		scheduledApp.Application.SvcNodes = clusterScheduledApp.FogSvcNodes
	} else {
		labelReq["app.kubernetes.io/version"] = *clusterScheduledApp.EdgeSvcVersion

		scheduledApp.Application.Svc = clusterScheduledApp.EdgeSvc
		scheduledApp.Application.SvcVersion = clusterScheduledApp.EdgeSvcVersion
		scheduledApp.Application.SvcNodes = clusterScheduledApp.EdgeSvcNodes
	}

	scheduledApp.Application.Labels = &labelReq

	return scheduledApp, nil
}

func (s *Service) ClusterApplicationStatus(ctx context.Context, d *dto.ClusterApplicationStatus) error {
	ctr := &domain.ApplicationFetchCtr{
		ApplicationID: &d.ApplicationID,
	}

	application, err := s.kbSvc.GetApplication(ctx, ctr)
	if err != nil {
		return err
	}

	if application == nil {
		return domain.ErrApplicationNotFound
	}

	now := time.Now().UTC()
	state := string(d.Status)

	if application.CloudSvcCluster != nil && *application.CloudSvcCluster == d.ClusterID {
		if *application.CloudSvcVersion == d.ApplicationVersion {
			application.CloudSvcStatus = &state
			application.CloudSvcHeartbeat = &now
		} else {
			return domain.ErrApplicationNotSynced
		}
	} else if application.FogSvcCluster != nil && *application.FogSvcCluster == d.ClusterID {
		if *application.FogSvcVersion == d.ApplicationVersion {
			application.FogSvcStatus = &state
			application.FogSvcHeartbeat = &now
		} else {
			return domain.ErrApplicationNotSynced
		}
	} else if application.EdgeSvcCluster != nil && *application.EdgeSvcCluster == d.ClusterID {
		if *application.EdgeSvcVersion == d.ApplicationVersion {
			application.EdgeSvcStatus = &state
			application.EdgeSvcHeartbeat = &now
		} else {
			return domain.ErrApplicationNotSynced
		}
	} else {
		return domain.ErrApplicationNotFound
	}

	if _, err := s.kbSvc.UpdateApplication(ctx, application); err != nil {
		return err
	}

	return nil
}

func (s *Service) SchedulePendingApps() error {
	if !s.Isleader() {
		spew.Dump("schedule pending apps - not a leader - returning")
		return nil
	}

	ctx := context.Background()

	applications, err := s.kbSvc.ListApplications(ctx, &domain.ApplicationListCtr{})
	if err != nil {
		return err
	}

	if len(applications) == 0 {
		log.Println("no applications found ...")
		return nil
	}

	clusters, err := s.kbSvc.ListClusters(ctx, &domain.ClusterListCtr{})
	if err != nil {
		return err
	}

	spew.Dump("schedule pending apps - ", clusters)

	var cloudClusters, fogClusters, edgeClusters []*domain.Cluster
	for _, cluster := range clusters {
		switch cluster.Type {
		case domain.ClusterTypeCloud:
			cloudClusters = append(cloudClusters, cluster)
		case domain.ClusterTypeFog:
			fogClusters = append(fogClusters, cluster)
		case domain.ClusterTypeEdge:
			edgeClusters = append(edgeClusters, cluster)
		}
	}

	fetchNodes := func(clusters []*domain.Cluster) ([]*domain.Node, error) {
		var nodes []*domain.Node
		for _, cluster := range clusters {
			clusterID := cluster.ID
			clusterNodes, err := s.kbSvc.ListNodes(ctx, &domain.NodeListCtr{
				ClusterID: &clusterID,
			})
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, clusterNodes...)
		}
		return nodes, nil
	}

	cloudNodes, err := fetchNodes(cloudClusters)
	if err != nil {
		return err
	}

	spew.Dump("schedule pending apps - cloud nodes", cloudNodes)

	fogNodes, err := fetchNodes(fogClusters)
	if err != nil {
		return err
	}

	spew.Dump("schedule pending apps - fog nodes", fogNodes)

	edgeNodes, err := fetchNodes(edgeClusters)
	if err != nil {
		return err
	}

	spew.Dump("schedule pending apps - edge nodes", edgeNodes)

	// For each application, try to assign clusters and nodes.
	// Then update the application only if for every non-nil service a cluster has been assigned.
	for i, app := range applications {
		// CloudSvc assignment.
		if app.CloudSvc != nil && app.CloudSvcCluster == nil {
			clusterID, nodes := s.scorerSvc.ScoreAndFilterNodes(cloudNodes, *app.QoS)
			if clusterID != "" && len(nodes) > 0 {
				nodesStr := strings.Join(nodes, ",")
				applications[i].CloudSvcCluster = &clusterID
				applications[i].CloudSvcNodes = &nodesStr
			}
		}

		// FogSvc assignment.
		if app.FogSvc != nil && app.FogSvcCluster == nil {
			clusterID, nodes := s.scorerSvc.ScoreAndFilterNodes(fogNodes, *app.QoS)
			if clusterID != "" && len(nodes) > 0 {
				nodesStr := strings.Join(nodes, ",")
				applications[i].FogSvcCluster = &clusterID
				applications[i].FogSvcNodes = &nodesStr
			}
		}

		// EdgeSvc assignment.
		if app.EdgeSvc != nil && app.EdgeSvcCluster == nil {
			clusterID, nodes := s.scorerSvc.ScoreAndFilterNodes(edgeNodes, *app.QoS)
			if clusterID != "" && len(nodes) > 0 {
				nodesStr := strings.Join(nodes, ",")
				applications[i].EdgeSvcCluster = &clusterID
				applications[i].EdgeSvcNodes = &nodesStr
			}
		}

		// Only update the application if every non-nil service has its cluster assigned.
		shouldUpdate := true
		if app.CloudSvc != nil && applications[i].CloudSvcCluster == nil {
			shouldUpdate = false
		}
		if app.FogSvc != nil && applications[i].FogSvcCluster == nil {
			shouldUpdate = false
		}
		if app.EdgeSvc != nil && applications[i].EdgeSvcCluster == nil {
			shouldUpdate = false
		}

		if shouldUpdate {
			_, err := s.kbSvc.UpdateApplication(ctx, applications[i])
			if err != nil {
				log.Println("error updating application", applications[i].ID, ":", err)
			}
		}
	}

	return nil
}

func (s *Service) HandleStalledApps() error {
	if !s.Isleader() {
		spew.Dump("handle stalled apps - not a leader - returning")
		return nil
	}

	now := time.Now().UTC()
	ctx := context.Background()

	apps, err := s.kbSvc.ListApplications(ctx, &domain.ApplicationListCtr{})
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		log.Println("no applications found ...")
		return nil
	}

	var stalledApps []*domain.Application
	for _, app := range apps {
		isUpdated := false

		if app.CloudSvcStatus != nil && domain.ApplicationState(*app.CloudSvcStatus) != domain.StateFailed && domain.ApplicationState(*app.CloudSvcStatus) != domain.StateProgressing && app.CloudSvcHeartbeat.Before(now.Add(-1*config.Get().App.ApplicationHeartbeatTimeout)) {
			spew.Dump("cloud svc older heartbeat ...")
			app.CloudSvcCluster = nil
			app.CloudSvcNodes = nil
			app.CloudSvcStatus = nil
			app.CloudSvcHeartbeat = nil

			isUpdated = true
		}

		if app.FogSvcStatus != nil && domain.ApplicationState(*app.FogSvcStatus) != domain.StateFailed && domain.ApplicationState(*app.CloudSvcStatus) != domain.StateProgressing && app.FogSvcHeartbeat.Before(now.Add(-1*config.Get().App.ApplicationHeartbeatTimeout)) {
			spew.Dump("fog svc older heartbeat ...")
			app.FogSvcCluster = nil
			app.FogSvcNodes = nil
			app.FogSvcStatus = nil
			app.FogSvcHeartbeat = nil

			isUpdated = true
		}

		if app.EdgeSvcStatus != nil && domain.ApplicationState(*app.EdgeSvcStatus) != domain.StateFailed && domain.ApplicationState(*app.CloudSvcStatus) != domain.StateProgressing && app.EdgeSvcHeartbeat.Before(now.Add(-1*config.Get().App.ApplicationHeartbeatTimeout)) {
			spew.Dump("edge svc older heartbeat ...")
			app.EdgeSvcCluster = nil
			app.EdgeSvcNodes = nil
			app.EdgeSvcStatus = nil
			app.EdgeSvcHeartbeat = nil

			isUpdated = true
		}

		if isUpdated {
			stalledApps = append(stalledApps, app)
		}
	}

	if len(stalledApps) == 0 {
		log.Println("no stalled apps found ...")
		return nil
	}

	for _, app := range stalledApps {
		_, err := s.kbSvc.UpdateApplication(ctx, app)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}
