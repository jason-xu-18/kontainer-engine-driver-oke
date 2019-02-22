package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/rancher/kontainer-engine/drivers/options"
	"github.com/rancher/kontainer-engine/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	runningStatus = "RUNNING"
)

var EnvMutex sync.Mutex

// Driver defines the struct of gke driver
type Driver struct {
	driverCapabilities types.Capabilities
}

type state struct {

	//the OCID of the tenancy
	TenancyID string `json:"tenancyID"`
	//the OCID of the user
	UserID string `json:"userID"`
	//resource region
	Region string `json:"region"`
	//the cluster compartment in which the cluster exists
	ClusterCompartment string `json:"clusterCompartment"`
	//the version of Kubernetes specified when creating the managed cluster
	KubernetesVersion string `json:"kubernetesVersion"`
	//the network compartment
	NetworkCompartment string `json:"networkCompartment"`
	//the name of the virtual cloud network
	Vcn string `json:"vcn"`
	//subnets1 in the virtual network to use
	Subnets1 string `json:"subnets1"`
	//subnets2 in the virtual network to use
	Subnets2 string `json:"subnets2"`
	//the CIDR block for Kubernetes services
	ServicesCidr string `json:"serviceCidr"`
	//The CIDR block for Kubernetes pods
	PodsCidr string `json:"podsCidr"`
	//The name of the node pool
	NodePoolName string `json:"nodePoolName"`
	//The version of Kubernetes running on the nodes in the node pool
	KubernetesVersionNode string `json:"kubernetesVersionNode"`
	//the image running on the nodes in the node pool
	NodeImageName string `json:"nodeImageName"`
	//the node shape of the nodes in the node pool
	NodeShape string `json:"nodeShape"`
	//the subnets in which to place nodes for node pool
	NodeSubnets string `json:"nodeSubnets"`
	//The number of nodes in each subnet
	QuantityPerSubnet string `json:"quantityPerSubnet"`
	//The SSH public key to access your nodes
	NodeSSHKey string `json:"nodeSshKey,omitempty"`
	//The api key of the user
	APIKey string `json:"apiKey,omitempty"`
	//The api key of the user
	FingerPrint string `json:"fingerPrint,omitempty"`
	//The map of Kubernetes labels (key/value pairs) to be applied to each node
	Labels map[string]string `json:"key,value"`

	// Cluster Name
	Name string `json:"name,omitempty"`

	// Cluster Id
	ID string `json:"cluster_id,omitempty"`

	// Cluster info
	ClusterInfo types.ClusterInfo

	types.UnimplementedClusterSizeAccess
}

func NewDriver() types.Driver {
	driver := &Driver{
		driverCapabilities: types.Capabilities{
			Capabilities: make(map[int64]bool),
		},
	}

	driver.driverCapabilities.AddCapability(types.GetVersionCapability)
	driver.driverCapabilities.AddCapability(types.SetVersionCapability)

	return driver
}

// GetDriverCreateOptions implements driver interface
func (d *Driver) GetDriverCreateOptions(ctx context.Context) (*types.DriverFlags, error) {
	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}
	driverFlag.Options["tenancy-ocid"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the OCID of the tenancy",
	}
	driverFlag.Options["user-ocid"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the OCID of the user",
	}
	driverFlag.Options["region"] = &types.Flag{
		Type:  types.StringType,
		Usage: "resource region",
		Default: &types.Default{
			DefaultString: "us-phoenix-1",
		},
	}
	driverFlag.Options["name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the name of the cluster that should be displayed to the user",
	}
	driverFlag.Options["cluster-compartment"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the cluster compartment in which the cluster exists",
	}
	driverFlag.Options["kubernetes-version"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The version of Kubernetes specified when creating the managed cluster",
		Default: &types.Default{
			DefaultString: "v1.11.5",
		},
	}
	driverFlag.Options["network-compartment"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the network compartment",
	}
	driverFlag.Options["vcn"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the name of the virtual cloud network",
	}
	driverFlag.Options["subnets1"] = &types.Flag{
		Type:  types.StringType,
		Usage: "subnets1 in the virtual network to use",
	}
	driverFlag.Options["subnets2"] = &types.Flag{
		Type:  types.StringType,
		Usage: "subnets2 in the virtual network to use",
	}
	driverFlag.Options["subnets"] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "comma-separated list of subnets in the virtual network to use,just support two subnets",
	}
	driverFlag.Options["service-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the CIDR block for Kubernetes services",
		Default: &types.Default{
			DefaultString: "10.96.0.0/16",
		},
	}
	driverFlag.Options["pods-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the CIDR block for Kubernetes pods",
		Default: &types.Default{
			DefaultString: "10.244.0.0/16",
		},
	}
	driverFlag.Options["nodepool-name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the name of the node pool",
		Default: &types.Default{
			DefaultString: "pool1",
		},
	}
	driverFlag.Options["kubernetes-versionnode"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the version of Kubernetes running on the nodes in the node pool",
		Default: &types.Default{
			DefaultString: "v1.11.5",
		},
	}
	driverFlag.Options["node-image"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the image running on the nodes in the node pool",
		Default: &types.Default{
			DefaultString: "Oracle-Linux-7.5",
		},
	}
	driverFlag.Options["node-shape"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the node shape of the nodes in the node pool",
		Default: &types.Default{
			DefaultString: "VM.Standard2.2",
		},
	}
	driverFlag.Options["node-subnets"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the subnets in which to place nodes for node pool",
	}
	driverFlag.Options["quantity-persubnet"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the number of nodes in each subnet",
		Default: &types.Default{
			DefaultString: "1",
		},
	}
	driverFlag.Options["ssh-key"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The SSH public key to access your nodes",
	}
	driverFlag.Options["api-key"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The api key of the user",
	}
	driverFlag.Options["finger-print"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The finger print of the user",
	}
	driverFlag.Options["labels"] = &types.Flag{
		Type:  types.StringSliceType,
		Usage: "The map of Kubernetes labels (key/value pairs) to be applied to each node",
	}

	return &driverFlag, nil
}

// GetDriverUpdateOptions implements driver interface
func (d *Driver) GetDriverUpdateOptions(ctx context.Context) (*types.DriverFlags, error) {
	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}
	driverFlag.Options["kubernetes-version"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Version of Kubernetes specified when creating the managed cluster",
		Default: &types.Default{
			DefaultString: "v1.11.5",
		},
	}
	return &driverFlag, nil
}

// SetDriverOptions implements driver interface
func getStateFromOpts(driverOptions *types.DriverOptions) (state, error) {
	d := state{
		ClusterInfo: types.ClusterInfo{
			Metadata: map[string]string{},
		},
	}
	d.ID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "cluster-id").(string)
	d.TenancyID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "tenancy-ocid").(string)
	d.UserID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "user-ocid").(string)
	d.Region = options.GetValueFromDriverOptions(driverOptions, types.StringType, "region").(string)
	d.Name = options.GetValueFromDriverOptions(driverOptions, types.StringType, "name").(string)
	d.ClusterCompartment = options.GetValueFromDriverOptions(driverOptions, types.StringType, "cluster-compartment").(string)
	d.KubernetesVersion = options.GetValueFromDriverOptions(driverOptions, types.StringType, "kubernetes-version").(string)
	d.NetworkCompartment = options.GetValueFromDriverOptions(driverOptions, types.StringType, "network-compartment").(string)
	d.Vcn = options.GetValueFromDriverOptions(driverOptions, types.StringType, "vcn").(string)
	d.Subnets1 = options.GetValueFromDriverOptions(driverOptions, types.StringType, "subnets1").(string)
	d.Subnets2 = options.GetValueFromDriverOptions(driverOptions, types.StringType, "subnets2").(string)
	d.ServicesCidr = options.GetValueFromDriverOptions(driverOptions, types.StringType, "service-cidr").(string)
	d.PodsCidr = options.GetValueFromDriverOptions(driverOptions, types.StringType, "pods-cidr").(string)
	d.NodePoolName = options.GetValueFromDriverOptions(driverOptions, types.StringType, "nodepool-name").(string)
	d.KubernetesVersionNode = options.GetValueFromDriverOptions(driverOptions, types.StringType, "kubernetes-versionnode").(string)
	d.NodeImageName = options.GetValueFromDriverOptions(driverOptions, types.StringType, "node-image").(string)
	d.NodeShape = options.GetValueFromDriverOptions(driverOptions, types.StringType, "node-shape").(string)
	d.NodeSubnets = options.GetValueFromDriverOptions(driverOptions, types.StringType, "node-subnets").(string)
	d.QuantityPerSubnet = options.GetValueFromDriverOptions(driverOptions, types.StringType, "quantity-persubnet").(string)
	d.NodeSSHKey = options.GetValueFromDriverOptions(driverOptions, types.StringType, "ssh-key").(string)
	d.APIKey = options.GetValueFromDriverOptions(driverOptions, types.StringType, "api-key").(string)
	d.FingerPrint = options.GetValueFromDriverOptions(driverOptions, types.StringType, "finger-print").(string)
	d.Labels = map[string]string{}
	labelValues := options.GetValueFromDriverOptions(driverOptions, types.StringSliceType, "labels").(*types.StringSlice)
	for _, part := range labelValues.Value {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			d.Labels[kv[0]] = kv[1]
		}
	}
	return d, d.validate()
}

func (s *state) validate() error {
	if s.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	if s.TenancyID == "" {
		return fmt.Errorf("Tenancy ID is required")
	}

	if s.UserID == "" {
		return fmt.Errorf("User ID is required")
	}

	if s.Region == "" {
		return fmt.Errorf("Region is required")
	}

	if s.ClusterCompartment == "" {
		return fmt.Errorf("Cluster Compartment OCID is required")
	}

	if s.APIKey == "" {
		return fmt.Errorf("Api key is required")
	}
	return nil
}

// Create implements driver interface
func (d *Driver) Create(ctx context.Context, opts *types.DriverOptions, _ *types.ClusterInfo) (*types.ClusterInfo, error) {
	logrus.Infof("Starting create")

	state, err := getStateFromOpts(opts)
	if err != nil {
		return nil, fmt.Errorf("error parsing state: %v", err)
	}

	logrus.Infof("tenancy: %s", state.TenancyID)
	logrus.Infof("user: %s", state.UserID)
	logrus.Infof("region: %s", state.Region)
	logrus.Infof("finerPrint: %s", state.FingerPrint)
	logrus.Infof("apikey: %s", state.APIKey)
	provider := common.NewRawConfigurationProvider(state.TenancyID, state.UserID, state.Region, state.FingerPrint, state.APIKey, nil)

	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("error creating identity client: %v", err)
	}

	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("error creating Container Engine client: %v", err)
	}

	request := identity.ListAvailabilityDomainsRequest{
		CompartmentId: &state.TenancyID,
	}

	ads, err := identityClient.ListAvailabilityDomains(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error listing available domain: %v", err)
	}

	logrus.Infof("List of available domains: %v", ads.Items)

	//create vcn
	vcn, err := CreateOrGetVcn(ctx, provider, state)
	if err != nil {
		return nil, fmt.Errorf("error creating vcn: %v", err)
	}

	compartmentID := state.NetworkCompartment
	if state.NetworkCompartment == "" {
		compartmentID = state.ClusterCompartment
	}
	vcnID := *vcn.Id
	kubernetesVersion := state.KubernetesVersion
	subnets1 := state.Subnets1
	subnets2 := state.Subnets2
	nodeSubnets := state.NodeSubnets
	subnet1CIDR := common.String("10.0.0.0/24")
	dnsLabel1 := common.String("subnetdns1")
	subnet1, err := CreateOrGetSubnetWithDetails(ctx, provider, compartmentID, vcnID, common.String(subnets1), subnet1CIDR, dnsLabel1, ads.Items[0].Name)
	if err != nil {
		return nil, fmt.Errorf("error create subnet1: %v", err)
	}
	logrus.Infof("create subnet1 complete")
	logrus.Infof("subnet1:", subnet1)

	subnet2CIDR := common.String("10.0.1.0/24")
	dnsLabel2 := common.String("subnetdns2")
	subnet2, err := CreateOrGetSubnetWithDetails(ctx, provider, compartmentID, vcnID, common.String(subnets2), subnet2CIDR, dnsLabel2, ads.Items[1].Name)
	if err != nil {
		return nil, fmt.Errorf("error create subnet1: %v", err)
	}
	logrus.Infof("create subnet2 complete")
	logrus.Infof("subnet2:", subnet2)

	subnet3CIDR := common.String("10.0.2.0/24")
	dnsLabel3 := common.String("subnetdns3")
	subnet3, err := CreateOrGetSubnetWithDetails(ctx, provider, compartmentID, vcnID, common.String(nodeSubnets), subnet3CIDR, dnsLabel3, ads.Items[2].Name)
	if err != nil {
		return nil, fmt.Errorf("error create subnet1: %v", err)
	}
	logrus.Infof("create subnet3 complete")
	logrus.Infof("subnet3:", subnet3)

	logrus.Infof("creating cluster")
	createClusterResp, err := createCluster(ctx, containerEngineClient, state.Name, compartmentID, vcnID, kubernetesVersion, *subnet1.Id, *subnet2.Id)
	if err != nil {
		return nil, fmt.Errorf("error create cluster: %v", err)
	}

	// wait until work request complete
	workReqResp, err := waitUntilWorkRequestComplete(containerEngineClient, createClusterResp.OpcWorkRequestId)
	if err != nil {
		return nil, fmt.Errorf("error wait request complete: %v", err)
	}
	logrus.Infof("cluster created")
	clusterID := getResourceID(workReqResp.Resources, containerengine.WorkRequestResourceActionTypeCreated, "CLUSTER")

	state.ID = *clusterID

	// create NodePool
	createNodePoolReq := containerengine.CreateNodePoolRequest{}
	createNodePoolReq.CompartmentId = common.String(compartmentID)
	createNodePoolReq.Name = common.String(state.Name)
	createNodePoolReq.ClusterId = clusterID
	createNodePoolReq.KubernetesVersion = common.String(state.KubernetesVersionNode)
	createNodePoolReq.NodeImageName = common.String(state.NodeImageName)
	createNodePoolReq.NodeShape = common.String(state.NodeShape)
	createNodePoolReq.SubnetIds = []string{*subnet3.Id}
	createNodePoolReq.InitialNodeLabels = []containerengine.KeyValue{}
	for key, value := range state.Labels {
		tmpKey := key
		tmpValue := value
		keyValue := containerengine.KeyValue{Key: &tmpKey, Value: &tmpValue}
		createNodePoolReq.InitialNodeLabels = append(createNodePoolReq.InitialNodeLabels, keyValue)
	}

	createNodePoolResp, err := containerEngineClient.CreateNodePool(ctx, createNodePoolReq)
	if err != nil {
		return nil, fmt.Errorf("error creating node pool: %v", err)
	}
	logrus.Infof("creating nodepool")

	workReqResp, err = waitUntilWorkRequestComplete(containerEngineClient, createNodePoolResp.OpcWorkRequestId)
	if err != nil {
		return nil, fmt.Errorf("error wait request complete: %v", err)
	}
	logrus.Infof("nodepool created")

	info := &types.ClusterInfo{}
	return info, storeState(info, state)
}

func storeState(info *types.ClusterInfo, state state) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if info.Metadata == nil {
		info.Metadata = map[string]string{}
	}
	info.Metadata["state"] = string(bytes)
	return nil
}

func getState(info *types.ClusterInfo) (state, error) {
	state := state{}
	// ignore error
	err := json.Unmarshal([]byte(info.Metadata["state"]), &state)
	return state, err
}

// Update implements driver interface
func (d *Driver) Update(ctx context.Context, info *types.ClusterInfo, opts *types.DriverOptions) (*types.ClusterInfo, error) {
	logrus.Infof("Starting update")

	state, err := getState(info)
	if err != nil {
		return nil, fmt.Errorf("error parsing state: %v", err)
	}

	newState, err := getStateFromOpts(opts)
	if err != nil {
		return nil, fmt.Errorf("error parsing state: %v", err)
	}

	logrus.Infof("tenancy: %s", newState.TenancyID)
	logrus.Infof("user: %s", newState.UserID)
	logrus.Infof("region: %s", newState.Region)
	logrus.Infof("finerPrint: %s", newState.FingerPrint)
	logrus.Infof("apikey: %s", newState.APIKey)
	provider := common.NewRawConfigurationProvider(newState.TenancyID, newState.UserID, newState.Region, newState.FingerPrint, newState.APIKey, nil)

	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("error creating Container Engine client: %v", err)
	}

	updateReq := containerengine.UpdateClusterRequest{}
	updateReq.ClusterId = common.String(newState.ID)
	// getReq := containerengine.GetClusterRequest{
	// 	ClusterId: updateReq.ClusterId,
	// }
	// _, err := containerEngineClient.GetCluster(ctx, getReq)
	if newState.Name != "" {
		updateReq.Name = common.String(newState.Name)
	}
	if newState.KubernetesVersion != "" {
		updateReq.KubernetesVersion = common.String(newState.KubernetesVersion)
	}

	updateResp, err := containerEngineClient.UpdateCluster(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("error update cluster: %v", err)
	}
	logrus.Infof("updating cluster")

	// wait until update complete
	waitUntilWorkRequestComplete(containerEngineClient, updateResp.OpcWorkRequestId)
	// if err != nil {
	// 	return nil, fmt.Errorf("error wait request complete: %v", err)
	// }
	logrus.Infof("cluster updated")
	state.Name = newState.Name
	state.ID = newState.ID
	state.KubernetesVersion = newState.KubernetesVersion
	return info, storeState(info, state)
}

func (d *Driver) PostCheck(ctx context.Context, info *types.ClusterInfo) (*types.ClusterInfo, error) {
	state, err := getState(info)
	if err != nil {
		return nil, err
	}

	logrus.Infof("tenancy: %s", state.TenancyID)
	logrus.Infof("user: %s", state.UserID)
	logrus.Infof("region: %s", state.Region)
	logrus.Infof("finerPrint: %s", state.FingerPrint)
	logrus.Infof("apikey: %s", state.APIKey)
	provider := common.NewRawConfigurationProvider(state.TenancyID, state.UserID, state.Region, state.FingerPrint, state.APIKey, nil)

	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("error creating Container Engine client: %v", err)
	}

	getReq := containerengine.GetClusterRequest{
		ClusterId: common.String(state.ID),
	}

	cluster, err := containerEngineClient.GetCluster(ctx, getReq)
	if err != nil {
		return nil, err
	}

	info.Endpoint = *cluster.Endpoints.Kubernetes
	info.Version = *cluster.KubernetesVersion
	//	info.Username = cluster.MasterAuth.Username
	//	info.Password = cluster.MasterAuth.Password
	//	info.RootCaCertificate = cluster.MasterAuth.ClusterCaCertificate
	//	info.ClientCertificate = cluster.MasterAuth.ClientCertificate
	//	info.ClientKey = cluster.MasterAuth.ClientKey
	//	info.NodeCount = cluster.CurrentNodeCount
	//	info.Metadata["nodePool"] = cluster.NodePools[0].Name
	//	serviceAccountToken, err := generateServiceAccountTokenForGke(cluster)
	//	if err != nil {
	//		return nil, err
	//	}
	//	info.ServiceAccountToken = serviceAccountToken
	return info, nil
}

// Remove implements driver interface
func (d *Driver) Remove(ctx context.Context, info *types.ClusterInfo) error {
	state, err := getState(info)
	if err != nil {
		return err
	}

	logrus.Infof("tenancy: %s", state.TenancyID)
	logrus.Infof("user: %s", state.UserID)
	logrus.Infof("region: %s", state.Region)
	logrus.Infof("finerPrint: %s", state.FingerPrint)
	logrus.Infof("apikey: %s", state.APIKey)
	provider := common.NewRawConfigurationProvider(state.TenancyID, state.UserID, state.Region, state.FingerPrint, state.APIKey, nil)

	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return fmt.Errorf("error creating Container Engine client: %v", err)
	}

	deleteReq := containerengine.DeleteClusterRequest{
		ClusterId: common.String(state.ID),
	}

	containerEngineClient.DeleteCluster(ctx, deleteReq)

	logrus.Infof("deleting cluster")

	return nil
}

func (d *Driver) GetClusterSize(ctx context.Context, info *types.ClusterInfo) (*types.NodeCount, error) {
	logrus.Info("get Kubernete version")

	state, err := getState(info)
	if err != nil {
		return nil, fmt.Errorf("error parsing state: %v", err)
	}

	logrus.Infof("tenancy: %s", state.TenancyID)
	logrus.Infof("user: %s", state.UserID)
	logrus.Infof("region: %s", state.Region)
	logrus.Infof("finerPrint: %s", state.FingerPrint)
	logrus.Infof("apikey: %s", state.APIKey)
	// provider := common.NewRawConfigurationProvider(state.TenancyID, state.UserID, state.Region, state.FingerPrint, state.APIKey, nil)

	// containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	// if err != nil {
	// 	return nil, fmt.Errorf("error creating Container Engine client: %v", err)
	// }

	updateReq := containerengine.UpdateClusterRequest{}
	updateReq.ClusterId = common.String(state.ID)
	// getReq := containerengine.GetClusterRequest{
	// 	ClusterId: updateReq.ClusterId,
	// }
	// _, err := containerEngineClient.GetCluster(ctx, getReq)

	version := &types.NodeCount{Count: int64(0)}

	return version, nil
}

func (d *Driver) GetVersion(ctx context.Context, info *types.ClusterInfo) (*types.KubernetesVersion, error) {
	logrus.Info("get Kubernete version")

	state, err := getState(info)
	if err != nil {
		return nil, fmt.Errorf("error parsing state: %v", err)
	}

	logrus.Infof("tenancy: %s", state.TenancyID)
	logrus.Infof("user: %s", state.UserID)
	logrus.Infof("region: %s", state.Region)
	logrus.Infof("finerPrint: %s", state.FingerPrint)
	logrus.Infof("apikey: %s", state.APIKey)
	provider := common.NewRawConfigurationProvider(state.TenancyID, state.UserID, state.Region, state.FingerPrint, state.APIKey, nil)

	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("error creating Container Engine client: %v", err)
	}

	updateReq := containerengine.UpdateClusterRequest{}
	updateReq.ClusterId = common.String(state.ID)
	getReq := containerengine.GetClusterRequest{
		ClusterId: updateReq.ClusterId,
	}
	getResp, err := containerEngineClient.GetCluster(ctx, getReq)

	version := &types.KubernetesVersion{Version: *getResp.Cluster.KubernetesVersion}

	return version, nil
}

func (d *Driver) SetVersion(ctx context.Context, info *types.ClusterInfo, version *types.KubernetesVersion) error {
	logrus.Info("updating Kubernete version")

	state, err := getState(info)
	if err != nil {
		return fmt.Errorf("error parsing state: %v", err)
	}

	logrus.Infof("tenancy: %s", state.TenancyID)
	logrus.Infof("user: %s", state.UserID)
	logrus.Infof("region: %s", state.Region)
	logrus.Infof("finerPrint: %s", state.FingerPrint)
	logrus.Infof("apikey: %s", state.APIKey)
	provider := common.NewRawConfigurationProvider(state.TenancyID, state.UserID, state.Region, state.FingerPrint, state.APIKey, nil)

	containerEngineClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(provider)
	if err != nil {
		return fmt.Errorf("error creating Container Engine client: %v", err)
	}

	updateReq := containerengine.UpdateClusterRequest{}
	updateReq.ClusterId = common.String(state.ID)
	// getReq := containerengine.GetClusterRequest{
	// 	ClusterId: updateReq.ClusterId,
	// }
	// _, err := containerEngineClient.GetCluster(ctx, getReq)

	if state.KubernetesVersion != "" {
		updateReq.KubernetesVersion = common.String(version.Version)
	}

	updateResp, err := containerEngineClient.UpdateCluster(ctx, updateReq)
	if err != nil {
		return fmt.Errorf("error update cluster: %v", err)
	}
	logrus.Infof("updating cluster")

	// wait until update complete
	waitUntilWorkRequestComplete(containerEngineClient, updateResp.OpcWorkRequestId)
	// if err != nil {
	// 	return fmt.Errorf("error wait request complete: %v", err)
	// }
	logrus.Infof("cluster updated")
	state.KubernetesVersion = version.Version
	storeState(info, state)

	return nil
}

func (d *Driver) GetCapabilities(ctx context.Context) (*types.Capabilities, error) {
	return &d.driverCapabilities, nil
}

// CreateOrGetVcn either creates a new Virtual Cloud Network (VCN) or get the one already exist
func CreateOrGetVcn(ctx context.Context, provider common.ConfigurationProvider, state state) (core.Vcn, error) {
	c, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return core.Vcn{}, fmt.Errorf("error creating VirtualNetworkClient: %v", err)
	}

	compartmentID := common.String(state.NetworkCompartment)
	if state.NetworkCompartment == "" {
		compartmentID = common.String(state.ClusterCompartment)
	}

	vcnItems := listVcns(ctx, c, compartmentID)

	for _, element := range vcnItems {
		if *element.Id == state.Vcn {
			// VCN already created, return it
			return element, nil
		}
	}

	// create a new VCN
	request := core.CreateVcnRequest{}
	if state.ServicesCidr == "" {
		request.CidrBlock = common.String("10.96.0.0/16")
	} else {
		request.CidrBlock = common.String(state.ServicesCidr)
	}

	request.CompartmentId = compartmentID

	request.DisplayName = common.String("rancherVcn")
	request.DnsLabel = common.String("vcndns")

	r, err := c.CreateVcn(ctx, request)
	if err != nil {
		return core.Vcn{}, fmt.Errorf("error creating VCN: %v", err)
	}
	return r.Vcn, nil
}

func listVcns(ctx context.Context, c core.VirtualNetworkClient, CompartmentID *string) []core.Vcn {
	request := core.ListVcnsRequest{
		CompartmentId: CompartmentID,
	}

	r, err := c.ListVcns(ctx, request)
	if err != nil {
		fmt.Println("error list vcn:", err)
	}
	return r.Items
}

func listSubnets(ctx context.Context, c core.VirtualNetworkClient, provider common.ConfigurationProvider, CompartmentID *string, vcnID *string) []core.Subnet {

	request := core.ListSubnetsRequest{
		CompartmentId: CompartmentID,
		VcnId:         vcnID,
	}

	r, err := c.ListSubnets(ctx, request)
	if err != nil {
		fmt.Println("error list subnets:", err)
	}

	return r.Items
}

// CreateOrGetSubnetWithDetails either creates a new Virtual Cloud Network (VCN) or get the one already exist
// with detail info
func CreateOrGetSubnetWithDetails(ctx context.Context, provider common.ConfigurationProvider, CompartmentID string, vcnID string, subnetID *string, cidrBlock *string, dnsLabel *string, availableDomain *string) (core.Subnet, error) {
	c, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return core.Subnet{}, fmt.Errorf("error creating virtual network client: %v", err)
	}

	subnets := listSubnets(ctx, c, provider, common.String(CompartmentID), common.String(vcnID))

	// check if the subnet has already been created
	for _, element := range subnets {
		if *element.Id == *subnetID {
			// find the subnet, return it
			return element, nil
		}
	}

	// create a new subnet
	request := core.CreateSubnetRequest{}
	request.AvailabilityDomain = availableDomain
	request.CompartmentId = common.String(CompartmentID)
	request.CidrBlock = cidrBlock
	request.DisplayName = common.String("rancherSubnets")
	request.DnsLabel = dnsLabel
	request.RequestMetadata = GetRequestMetadataWithDefaultRetryPolicy()
	request.VcnId = common.String(vcnID)

	r, err := c.CreateSubnet(ctx, request)
	if err != nil {
		return core.Subnet{}, fmt.Errorf("error creating subnet: %v", err)
	}

	// retry condition check, stop unitl return true
	pollUntilAvailable := func(r common.OCIOperationResponse) bool {
		if converted, ok := r.Response.(core.GetSubnetResponse); ok {
			return converted.LifecycleState != core.SubnetLifecycleStateAvailable
		}
		return true
	}

	pollGetRequest := core.GetSubnetRequest{
		SubnetId:        r.Id,
		RequestMetadata: GetRequestMetadataWithCustomizedRetryPolicy(pollUntilAvailable),
	}

	// wait for lifecyle become running
	_, pollErr := c.GetSubnet(ctx, pollGetRequest)
	if pollErr != nil {
		return core.Subnet{}, fmt.Errorf("error creating subnet: %v", err)
	}

	// update the security rules
	getReq := core.GetSecurityListRequest{
		SecurityListId: common.String(r.SecurityListIds[0]),
	}

	getResp, err := c.GetSecurityList(ctx, getReq)
	if err != nil {
		return core.Subnet{}, fmt.Errorf("error creating subnet: %v", err)
	}

	// this security rule allows remote control the instance
	portRange := core.PortRange{
		Max: common.Int(1521),
		Min: common.Int(1521),
	}

	newRules := append(getResp.IngressSecurityRules, core.IngressSecurityRule{
		Protocol: common.String("6"), // TCP
		Source:   common.String("0.0.0.0/0"),
		TcpOptions: &core.TcpOptions{
			DestinationPortRange: &portRange,
		},
	})

	updateReq := core.UpdateSecurityListRequest{
		SecurityListId: common.String(r.SecurityListIds[0]),
	}

	updateReq.IngressSecurityRules = newRules

	_, err = c.UpdateSecurityList(ctx, updateReq)
	if err != nil {
		return core.Subnet{}, fmt.Errorf("error creating subnet: %v", err)
	}

	return r.Subnet, nil
}

// create a cluster
func createCluster(
	ctx context.Context,
	client containerengine.ContainerEngineClient,
	name, CompartmentID, vcnID, kubernetesVersion, subnet1ID, subnet2ID string) (containerengine.CreateClusterResponse, error) {
	req := containerengine.CreateClusterRequest{}
	req.Name = common.String(name)
	req.CompartmentId = common.String(CompartmentID)
	req.VcnId = common.String(vcnID)
	req.KubernetesVersion = common.String(kubernetesVersion)
	req.Options = &containerengine.ClusterCreateOptions{
		ServiceLbSubnetIds: []string{subnet1ID, subnet2ID},
	}

	resp, err := client.CreateCluster(ctx, req)
	if err != nil {
		return containerengine.CreateClusterResponse{}, fmt.Errorf("error creating cluster: %v", err)
	}

	return resp, nil
}

func GetRequestMetadataWithDefaultRetryPolicy() common.RequestMetadata {
	return common.RequestMetadata{
		RetryPolicy: getDefaultRetryPolicy(),
	}
}

func GetRequestMetadataWithCustomizedRetryPolicy(fn func(r common.OCIOperationResponse) bool) common.RequestMetadata {
	return common.RequestMetadata{
		RetryPolicy: getExponentialBackoffRetryPolicy(uint(20), fn),
	}
}

func getDefaultRetryPolicy() *common.RetryPolicy {
	// how many times to do the retry
	attempts := uint(10)

	// retry for all non-200 status code
	retryOnAllNon200ResponseCodes := func(r common.OCIOperationResponse) bool {
		return !(r.Error == nil && 199 < r.Response.HTTPResponse().StatusCode && r.Response.HTTPResponse().StatusCode < 300)
	}
	return getExponentialBackoffRetryPolicy(attempts, retryOnAllNon200ResponseCodes)
}

func getExponentialBackoffRetryPolicy(n uint, fn func(r common.OCIOperationResponse) bool) *common.RetryPolicy {
	// the duration between each retry operation, you might want to waite longer each time the retry fails
	exponentialBackoff := func(r common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(r.AttemptNumber-1))) * time.Second
	}
	policy := common.NewRetryPolicy(n, fn, exponentialBackoff)
	return &policy
}

// wait until work request finish
func waitUntilWorkRequestComplete(client containerengine.ContainerEngineClient, workReuqestID *string) (containerengine.GetWorkRequestResponse, error) {
	// retry GetWorkRequest call until TimeFinished is set
	shouldRetryFunc := func(r common.OCIOperationResponse) bool {
		return r.Response.(containerengine.GetWorkRequestResponse).TimeFinished == nil
	}

	getWorkReq := containerengine.GetWorkRequestRequest{
		WorkRequestId:   workReuqestID,
		RequestMetadata: GetRequestMetadataWithCustomizedRetryPolicy(shouldRetryFunc),
	}

	getResp, err := client.GetWorkRequest(context.Background(), getWorkReq)
	if err != nil {
		return containerengine.GetWorkRequestResponse{}, fmt.Errorf("error waiting work request complete: %v", err)
	}
	return getResp, nil
}

// getResourceID return a resource ID based on the filter of resource actionType and entityType
func getResourceID(resources []containerengine.WorkRequestResource, actionType containerengine.WorkRequestResourceActionTypeEnum, entityType string) *string {
	for _, resource := range resources {
		if resource.ActionType == actionType && strings.ToUpper(*resource.EntityType) == entityType {
			return resource.Identifier
		}
	}

	fmt.Println("cannot find matched resources")
	return nil
}

func (d *Driver) GetK8SCapabilities(ctx context.Context, opts *types.DriverOptions) (*types.K8SCapabilities, error) {
	logrus.Debug("GetK8SCapabilities unimplemented")
	return nil, nil
}

func (d *Driver) SetClusterSize(ctx context.Context, info *types.ClusterInfo, count *types.NodeCount) error {
	logrus.Debug("SetClusterSize unimplemented")
	return nil
}
