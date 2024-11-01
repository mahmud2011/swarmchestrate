package resourceagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/mahmud2011/swarmchestrate/resource-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-agent/domain/dto"
	"github.com/mahmud2011/swarmchestrate/resource-agent/internal/config"
)

const (
	KeyCloudClusterIP = "[CLOUD_CLUSTER_IP]"
	KeyFogClusterIP   = "[FOG_CLUSTER_IP]"
	KeyEdgeClusterIP  = "[EDGE_CLUSTER_IP]"
	KeyResourceLeads  = "resource-leads"
)

type IResourceLeadSvc interface {
	RegisterCluster(ctx context.Context, d *dto.AddClusterReq) (*domain.Cluster, error)
	SyncClusterConfig(ctx context.Context, resourceLeadHost string) (map[string]string, error)
	UpsertNodes(resourceLeadHost string, clusterID string, workerNodes []*dto.Node) error
	GetScheduledApps(resourceLeadHost string, clusterID string) ([]*dto.ScheduledApplication, error)
	NotifyAppHealth(resourceLeadHost string, ctr *dto.ClusterApplicationStatus) error
}

type Service struct {
	resourceLeadSvc IResourceLeadSvc
	clusterID       string
	mu              sync.RWMutex
}

func NewService(ms IResourceLeadSvc) *Service {
	return &Service{
		resourceLeadSvc: ms,
	}
}

// getClientset returns a Kubernetes clientset using in-cluster configuration.
func getClientset() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error building in-cluster config: %v", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}
	return cs, nil
}

// getResourceLeadIP retrieves a resource-lead IP from the "clusters" ConfigMap.
// If update is true, it updates the roundRobinIndex.
func getResourceLeadIP(ctx context.Context, clientset *kubernetes.Clientset, update bool) (string, error) {
	cm, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Get(ctx, "clusters", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get clusters ConfigMap: %v", err)
	}
	leadsStr, ok := cm.Data[KeyResourceLeads]
	if !ok || leadsStr == "" {
		return "", fmt.Errorf("resource-leads key not found or empty in clusters ConfigMap")
	}
	parts := strings.Split(leadsStr, ",")
	var valid []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return "", fmt.Errorf("no valid resource-lead found in clusters ConfigMap")
	}
	index := 0
	if idxStr, ok := cm.Data["roundRobinIndex"]; ok && idxStr != "" {
		if i, err := strconv.Atoi(idxStr); err == nil {
			index = i
		}
	}
	if index >= len(valid) {
		index = 0
	}
	selected := valid[index]
	if update {
		newIndex := (index + 1) % len(valid)
		cm.Data["roundRobinIndex"] = strconv.Itoa(newIndex)
		if _, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
			slog.Error("failed to update clusters ConfigMap", "error", err)
		}
	}
	leadData, ok := cm.Data[selected]
	if !ok || leadData == "" {
		return "", fmt.Errorf("selected resource-lead %s not found in clusters ConfigMap", selected)
	}
	var leadInfo struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal([]byte(leadData), &leadInfo); err != nil {
		return "", fmt.Errorf("error parsing resource-lead data for %s: %v", selected, err)
	}
	if leadInfo.IP == "" {
		return "", fmt.Errorf("no IP found for resource-lead %s", selected)
	}
	return leadInfo.IP, nil
}

func (s *Service) RegisterCluster(ctx context.Context, cfg *config.Config) error {
	clientset, err := getClientset()
	if err != nil {
		slog.Error("RegisterCluster: failed to get clientset", "error", err)
		return err
	}
	selfCM, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Get(context.TODO(), "self", metav1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			iSvc, err := clientset.CoreV1().
				Services("ingress-nginx").
				Get(context.Background(), "ingress-nginx-controller", metav1.GetOptions{})
			if err != nil {
				slog.Error("RegisterCluster: error getting ingress service", "error", err)
				return err
			}
			if len(iSvc.Status.LoadBalancer.Ingress) == 0 {
				return fmt.Errorf("no ingress IP found")
			}
			var ip string
			for _, lb := range iSvc.Status.LoadBalancer.Ingress {
				if lb.IP != "" {
					ip = lb.IP
					break
				}
			}
			req := &dto.AddClusterReq{
				Type: domain.ClusterType(cfg.App.Type),
				Role: domain.ClusterRoleWorker,
				IP:   ip,
			}
			createdCluster, err := s.resourceLeadSvc.RegisterCluster(ctx, req)
			if err != nil {
				slog.Error("RegisterCluster: error registering cluster", "error", err)
				return err
			}
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "self"},
				Data: map[string]string{
					"id":   createdCluster.ID,
					"role": string(createdCluster.Role),
				},
			}
			if _, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Create(context.TODO(), cm, metav1.CreateOptions{}); err != nil {
				slog.Error("RegisterCluster: error creating self ConfigMap", "error", err)
				return err
			}
			s.mu.Lock()
			s.clusterID = createdCluster.ID
			s.mu.Unlock()
		} else {
			slog.Error("RegisterCluster: error retrieving self ConfigMap", "error", err)
			return err
		}
	} else {
		if clusterID, ok := selfCM.Data["id"]; ok && clusterID != "" {
			s.mu.Lock()
			s.clusterID = clusterID
			s.mu.Unlock()
		}
	}
	if _, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Get(context.TODO(), "clusters", metav1.GetOptions{}); err != nil {
		if k8sErr.IsNotFound(err) {
			followers := []string{"INIT"}
			clustersCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "clusters"},
				Data: map[string]string{
					KeyResourceLeads: strings.Join(followers, ","),
					"INIT":           fmt.Sprintf("{\"ip\": \"%s\"}", cfg.RegisterSvc.Host),
				},
			}
			if _, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Create(ctx, clustersCM, metav1.CreateOptions{}); err != nil {
				slog.Error("RegisterCluster: error creating clusters ConfigMap", "error", err)
				return err
			}
		}
	}
	if _, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Get(context.TODO(), "applications", metav1.GetOptions{}); err != nil {
		if k8sErr.IsNotFound(err) {
			appCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "applications"},
				Data:       map[string]string{},
			}
			if _, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Create(ctx, appCM, metav1.CreateOptions{}); err != nil {
				slog.Error("RegisterCluster: error creating applications ConfigMap", "error", err)
				return err
			}
		}
	}
	return nil
}

func (s *Service) SyncClusterConfig(ctx context.Context) error {
	clientset, err := getClientset()
	if err != nil {
		slog.Error("SyncClusterConfig: failed to get clientset", "error", err)
		return err
	}
	ip, err := getResourceLeadIP(ctx, clientset, true)
	if err != nil {
		return err
	}
	newConfig, err := s.resourceLeadSvc.SyncClusterConfig(ctx, ip)
	if err != nil {
		return fmt.Errorf("error syncing cluster config from resource-lead: %v", err)
	}
	cm, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Get(ctx, "clusters", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting clusters ConfigMap: %v", err)
	}
	for k, v := range newConfig {
		cm.Data[k] = v
	}
	if _, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("error updating clusters ConfigMap: %v", err)
	}
	return nil
}

func (s *Service) CollectNodeInfo() ([]*dto.Node, error) {
	clientset, err := getClientset()
	if err != nil {
		slog.Error("CollectNodeInfo: failed to get clientset", "error", err)
		return nil, err
	}
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		slog.Error("CollectNodeInfo: error listing nodes", "error", err)
		return nil, err
	}
	var workerNodes []*dto.Node
	for _, node := range nodeList.Items {
		if _, exists := node.Labels["node-role.kubernetes.io/control-plane"]; exists {
			continue
		}
		workerNodes = append(workerNodes, &dto.Node{
			Name:     node.Name,
			Location: strings.Replace(node.Labels["swarmchestrate.tu-berlin.de/location"], "_", "/", 1),
			CPU:      int(node.Status.Allocatable.Cpu().Value()),
			CPUArch:  node.Status.NodeInfo.Architecture,
			Memory:   float64(node.Status.Allocatable.Memory().Value()),
			NetworkBandwidth: func() float64 {
				f, _ := strconv.ParseFloat(node.Labels["swarmchestrate.tu-berlin.de/bandwidth"], 64)
				return f
			}(),
			EphemeralStorage: float64(node.Status.Allocatable.StorageEphemeral().Value()),
			Energy: func() float64 {
				f, _ := strconv.ParseFloat(node.Labels["swarmchestrate.tu-berlin.de/energy"], 64)
				return f
			}(),
			Pricing: func() float64 {
				f, _ := strconv.ParseFloat(node.Labels["swarmchestrate.tu-berlin.de/price"], 64)
				return f
			}(),
			IsReady: getNodeCondition(
				node.Status.Conditions,
				corev1.NodeReady,
			).Status == corev1.ConditionTrue,
			IsSchedulable: !node.Spec.Unschedulable,
			IsPIDPressureExists: getNodeCondition(
				node.Status.Conditions,
				corev1.NodePIDPressure,
			).Status == corev1.ConditionTrue,
			IsMemoryPressureExists: getNodeCondition(
				node.Status.Conditions,
				corev1.NodeMemoryPressure,
			).Status == corev1.ConditionTrue,
			IsDiskPressureExists: getNodeCondition(
				node.Status.Conditions,
				corev1.NodeDiskPressure,
			).Status == corev1.ConditionTrue,
		})
	}
	return workerNodes, nil
}

func (s *Service) SendNodeInfo(ctx context.Context, nodes []*dto.Node) error {
	s.mu.RLock()
	clusterID := s.clusterID
	s.mu.RUnlock()
	if clusterID == "" {
		return fmt.Errorf("cluster not registered; clusterID is empty")
	}
	clientset, err := getClientset()
	if err != nil {
		slog.Error("SendNodeInfo: failed to get clientset", "error", err)
		return err
	}
	ip, err := getResourceLeadIP(ctx, clientset, true)
	if err != nil {
		return err
	}
	if err := s.resourceLeadSvc.UpsertNodes(ip, clusterID, nodes); err != nil {
		return fmt.Errorf("error sending node info to resource-lead %s: %v", ip, err)
	}
	return nil
}

func (s *Service) GetScheduledJobs(ctx context.Context) ([]*dto.ScheduledApplication, error) {
	s.mu.RLock()
	clusterID := s.clusterID
	s.mu.RUnlock()
	if clusterID == "" {
		return nil, fmt.Errorf("cluster not registered; clusterID is empty")
	}
	clientset, err := getClientset()
	if err != nil {
		return nil, fmt.Errorf("error getting clientset: %v", err)
	}
	ip, err := getResourceLeadIP(ctx, clientset, false)
	if err != nil {
		return nil, err
	}
	scheduledApps, err := s.resourceLeadSvc.GetScheduledApps(ip, clusterID)
	if err != nil {
		return nil, fmt.Errorf("error getting scheduled apps from resource-lead %s: %v", ip, err)
	}
	return scheduledApps, nil
}

func (s *Service) DeployScheduledJobs(scheduledApplications []*dto.ScheduledApplication) error {
	s.mu.RLock()
	clusterID := s.clusterID
	s.mu.RUnlock()
	if clusterID == "" {
		return fmt.Errorf("cluster not registered; clusterID is empty")
	}
	ctx := context.TODO()
	clientset, err := getClientset()
	if err != nil {
		slog.Error("DeployScheduledJobs: failed to get clientset", "error", err)
		return err
	}
	ip, err := getResourceLeadIP(ctx, clientset, false)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	mergeLabels := func(a, b map[string]string) map[string]string {
		if a == nil {
			a = make(map[string]string)
		}
		for k, v := range b {
			if _, exists := a[k]; !exists {
				a[k] = v
			}
		}
		return a
	}
	for _, app := range scheduledApplications {
		wg.Add(1)
		go func(app *dto.ScheduledApplication) {
			defer wg.Done()
			var errs []string
			var deployedKinds []string
			namespace := app.Application.Name
			if _, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); err != nil {
				if k8sErr.IsNotFound(err) {
					if _, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: namespace},
					}, metav1.CreateOptions{}); err != nil {
						errs = append(errs, err.Error())
					}
				} else {
					errs = append(errs, err.Error())
				}
			}
			appVersion := ""
			if app.Application.SvcVersion != nil {
				appVersion = *app.Application.SvcVersion
			}
			k8sResources := app.Application.Svc
			nodes := strings.Split(*app.Application.SvcNodes, ",")
			for _, resource := range k8sResources {
				kindVal, ok := resource["kind"]
				if !ok {
					errs = append(errs, "resource kind not found")
					continue
				}
				kind, ok := kindVal.(string)
				if !ok {
					errs = append(errs, "invalid resource kind")
					continue
				}
				switch kind {
				case "Service":
					resourceJSON, err := json.Marshal(resource)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					var svcObj corev1.Service
					if err := json.Unmarshal(resourceJSON, &svcObj); err != nil {
						errs = append(errs, err.Error())
						continue
					}
					svcObj.Labels = mergeLabels(svcObj.Labels, *app.Application.Labels)
					if _, err := clientset.CoreV1().Services(namespace).Get(ctx, svcObj.Name, metav1.GetOptions{}); err != nil {
						if k8sErr.IsNotFound(err) {
							if _, err := clientset.CoreV1().Services(namespace).Create(ctx, &svcObj, metav1.CreateOptions{}); err != nil {
								errs = append(errs, err.Error())
								continue
							}
						} else {
							errs = append(errs, err.Error())
							continue
						}
					} else {
						if _, err := clientset.CoreV1().Services(namespace).Update(ctx, &svcObj, metav1.UpdateOptions{}); err != nil {
							errs = append(errs, err.Error())
							continue
						}
					}
					deployedKinds = append(deployedKinds, "Service")
				case "ServiceAccount":
					resourceJSON, err := json.Marshal(resource)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					var sa corev1.ServiceAccount
					if err := json.Unmarshal(resourceJSON, &sa); err != nil {
						errs = append(errs, err.Error())
						continue
					}
					sa.Labels = mergeLabels(sa.Labels, *app.Application.Labels)
					if _, err := clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, sa.Name, metav1.GetOptions{}); err != nil {
						if k8sErr.IsNotFound(err) {
							if _, err := clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, &sa, metav1.CreateOptions{}); err != nil {
								errs = append(errs, err.Error())
								continue
							}
						} else {
							errs = append(errs, err.Error())
							continue
						}
					} else {
						if _, err := clientset.CoreV1().ServiceAccounts(namespace).Update(ctx, &sa, metav1.UpdateOptions{}); err != nil {
							errs = append(errs, err.Error())
							continue
						}
					}
					deployedKinds = append(deployedKinds, "ServiceAccount")
				case "ConfigMap":
					resourceJSON, err := json.Marshal(resource)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					var cmObj corev1.ConfigMap
					if err := json.Unmarshal(resourceJSON, &cmObj); err != nil {
						errs = append(errs, err.Error())
						continue
					}
					cmObj.Labels = mergeLabels(cmObj.Labels, *app.Application.Labels)
					for k, v := range cmObj.Data {
						if v == KeyCloudClusterIP {
							cmObj.Data[k] = *app.CloudIP
						} else if v == KeyFogClusterIP {
							cmObj.Data[k] = *app.FogIP
						} else if v == KeyEdgeClusterIP {
							cmObj.Data[k] = *app.EdgeIP
						}
					}
					if _, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, cmObj.Name, metav1.GetOptions{}); err != nil {
						if k8sErr.IsNotFound(err) {
							if _, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &cmObj, metav1.CreateOptions{}); err != nil {
								errs = append(errs, err.Error())
								continue
							}
						} else {
							errs = append(errs, err.Error())
							continue
						}
					} else {
						if _, err := clientset.CoreV1().ConfigMaps(namespace).Update(ctx, &cmObj, metav1.UpdateOptions{}); err != nil {
							errs = append(errs, err.Error())
							continue
						}
					}
					deployedKinds = append(deployedKinds, "ConfigMap")
				case "Ingress":
					resourceJSON, err := json.Marshal(resource)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					var ingObj networkingv1.Ingress
					if err := json.Unmarshal(resourceJSON, &ingObj); err != nil {
						errs = append(errs, err.Error())
						continue
					}
					ingObj.Labels = mergeLabels(ingObj.Labels, *app.Application.Labels)
					if _, err := clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingObj.Name, metav1.GetOptions{}); err != nil {
						if k8sErr.IsNotFound(err) {
							if _, err := clientset.NetworkingV1().Ingresses(namespace).Create(ctx, &ingObj, metav1.CreateOptions{}); err != nil {
								errs = append(errs, err.Error())
								continue
							}
						} else {
							errs = append(errs, err.Error())
							continue
						}
					} else {
						if _, err := clientset.NetworkingV1().Ingresses(namespace).Update(ctx, &ingObj, metav1.UpdateOptions{}); err != nil {
							errs = append(errs, err.Error())
							continue
						}
					}
					deployedKinds = append(deployedKinds, "Ingress")
				case "Deployment":
					resourceJSON, err := json.Marshal(resource)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					var deployObj appsv1.Deployment
					if err := json.Unmarshal(resourceJSON, &deployObj); err != nil {
						errs = append(errs, err.Error())
						continue
					}
					deployObj.Labels = mergeLabels(deployObj.Labels, *app.Application.Labels)
					deployObj.Spec.Template.Labels = mergeLabels(
						deployObj.Spec.Template.Labels,
						*app.Application.Labels,
					)
					deployObj.Spec.Template.Spec.Affinity = &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/hostname",
												Operator: corev1.NodeSelectorOpIn,
												Values:   nodes,
											},
										},
									},
								},
							},
						},
					}
					if _, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deployObj.Name, metav1.GetOptions{}); err != nil {
						if k8sErr.IsNotFound(err) {
							if _, err := clientset.AppsV1().Deployments(namespace).Create(ctx, &deployObj, metav1.CreateOptions{}); err != nil {
								errs = append(errs, err.Error())
								continue
							}
						} else {
							errs = append(errs, err.Error())
							continue
						}
					} else {
						if _, err := clientset.AppsV1().Deployments(namespace).Update(ctx, &deployObj, metav1.UpdateOptions{}); err != nil {
							errs = append(errs, err.Error())
							continue
						}
					}
					deployedKinds = append(deployedKinds, "Deployment")
				default:
					errs = append(errs, fmt.Sprintf("unknown resource kind: %s", kind))
				}
			}
			if len(errs) > 0 {
				slog.Error("DeployScheduledJobs: deployment errors", "errors", errs)
				notifyStatus := &dto.ClusterApplicationStatus{
					ApplicationID:      app.Application.ID,
					ClusterID:          clusterID,
					ApplicationVersion: appVersion,
					Status:             domain.StateFailed,
					Message:            strings.Join(errs, ";"),
				}
				if notifyErr := s.resourceLeadSvc.NotifyAppHealth(ip, notifyStatus); notifyErr != nil {
					slog.Error(
						"DeployScheduledJobs: error notifying app health",
						"appID", app.Application.ID,
						"error", notifyErr,
					)
				}
				return
			}
			successStatus := &dto.ClusterApplicationStatus{
				ApplicationID:      app.Application.ID,
				ClusterID:          clusterID,
				ApplicationVersion: appVersion,
				Status:             domain.StateProgressing,
				Message:            "Application resources deployed; status progressing",
			}
			if notifyErr := s.resourceLeadSvc.NotifyAppHealth(ip, successStatus); notifyErr != nil {
				slog.Error(
					"DeployScheduledJobs: error notifying app health",
					"appID", app.Application.ID,
					"error", notifyErr,
				)
			}
			appCfg := dto.ApplicationConfig{
				AppID:        app.Application.ID,
				AppVersion:   appVersion,
				K8sResources: strings.Join(deployedKinds, ","),
				DeployedAt:   time.Now().UTC(),
			}
			cfgJSON, err := json.Marshal(appCfg)
			if err != nil {
				slog.Error(
					"DeployScheduledJobs: error marshalling app config",
					"appID", app.Application.ID,
					"error", err,
				)
			}
			cmName := "applications"
			updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				cm, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Get(ctx, cmName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if cm.Data == nil {
					cm.Data = make(map[string]string)
				}
				cm.Data[app.Application.Name] = string(cfgJSON)
				_, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Update(ctx, cm, metav1.UpdateOptions{})
				return err
			})
			if updateErr != nil {
				slog.Error(
					"DeployScheduledJobs: error updating applications ConfigMap",
					"appID", app.Application.ID,
					"error", updateErr,
				)
			}
		}(app)
	}
	wg.Wait()
	return nil
}

func (s *Service) SendApplicationStatus() error {
	s.mu.RLock()
	clusterID := s.clusterID
	s.mu.RUnlock()
	if clusterID == "" {
		return fmt.Errorf("cluster not registered; clusterID is empty")
	}
	ctx := context.Background()
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("SendApplicationStatus: error getting clientset: %v", err)
	}
	ip, err := getResourceLeadIP(ctx, clientset, false)
	if err != nil {
		return err
	}
	appCM, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Get(ctx, "applications", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("SendApplicationStatus: failed to get applications ConfigMap: %v", err)
	}
	var wg sync.WaitGroup
	for appName, appConfig := range appCM.Data {
		wg.Add(1)
		go func(appName, appConfig string) {
			defer wg.Done()
			var appCfg dto.ApplicationConfig
			if err := json.Unmarshal([]byte(appConfig), &appCfg); err != nil {
				slog.Error("SendApplicationStatus: error unmarshalling app config", "appName", appName, "error", err)
				return
			}
			status := &dto.ClusterApplicationStatus{
				ApplicationID:      appCfg.AppID,
				ClusterID:          clusterID,
				ApplicationVersion: appCfg.AppVersion,
			}
			depList, err := clientset.AppsV1().Deployments(appName).List(ctx, metav1.ListOptions{})
			if err != nil {
				if k8sErr.IsNotFound(err) {
					status.Status = domain.StateFailed
					if err := s.resourceLeadSvc.NotifyAppHealth(ip, status); err != nil {
						if err == domain.ErrApplicationNotFound {
							err = s.Cleanup(appName)
							slog.Error("SendApplicationStatus: cleanup error", "appName", appName, "error", err)
						}
					}
				}
				return
			}
			allReady := true
			for _, dep := range depList.Items {
				if dep.Spec.Replicas != nil && dep.Status.ReadyReplicas < *dep.Spec.Replicas {
					allReady = false
					break
				}
			}
			elapsed := time.Since(appCfg.DeployedAt)
			if allReady {
				status.Status = domain.StateHealthy
			} else {
				if elapsed > config.Get().App.DeploymentTimeout {
					status.Status = domain.StateFailed
				} else {
					status.Status = domain.StateProgressing
				}
			}
			if err := s.resourceLeadSvc.NotifyAppHealth(ip, status); err != nil {
				if err == domain.ErrApplicationNotFound {
					err = s.Cleanup(appName)
					slog.Error("SendApplicationStatus: cleanup error", "appName", appName, "error", err)
					return
				}
				slog.Error("SendApplicationStatus: error notifying app health", "appName", appName, "error", err)
			}
		}(appName, appConfig)
	}
	wg.Wait()
	return nil
}

func (s *Service) Cleanup(name string) error {
	ctx := context.Background()
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("Cleanup: error getting clientset: %v", err)
	}
	if err := clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !k8sErr.IsNotFound(err) {
			return fmt.Errorf("Cleanup: error deleting namespace %s: %v", name, err)
		}
	}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, err := clientset.CoreV1().ConfigMaps("swarmchestrate").Get(ctx, "applications", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if cm.Data == nil {
			return nil
		}
		delete(cm.Data, name)
		_, err = clientset.CoreV1().ConfigMaps("swarmchestrate").Update(ctx, cm, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return fmt.Errorf("Cleanup: error updating applications ConfigMap: %v", err)
	}
	return nil
}

func getNodeCondition(conditions []corev1.NodeCondition, conditionType corev1.NodeConditionType) corev1.NodeCondition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition
		}
	}
	return corev1.NodeCondition{}
}
