package proxy

import (
	"fmt"
	"net"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/google/uuid"
)

const (
	NatTableName string = "nat"
	TCPProtocol  string = "tcp"

	PreroutingChainName               string = "PREROUTING"
	OutputChainName                   string = "OUTPUT"
	PostroutingChainName              string = "POSTROUTING"
	KuberboatServicesChainName        string = "KUBERBOAT-SERVICES"
	KuberboatPostroutingChainName     string = "KUBERBOAT-POSTROUTING"
	KuberboatHostPostroutingChainName string = "KUBERBOAT-HOST-POSTROUTING"
	KuberboatMarkChainName            string = "KUBERBOAT-MARK-MASQ"
	KuberboatHostMarkChainName        string = "KUBERBOAT-HOST-MARK-MASQ"
	KuberboatServiceChainPrefix       string = "KUBERBOAT-SVC-"
	KuberboatPodChainPrefix           string = "KUBERBOAT-SEP-"
	DNatChainName                     string = "DNAT"
	SNatChainName                     string = "SNAT"
	MasqueradeTargetName              string = "MASQUERADE"
	MarkTargetName                    string = "MARK"
	ReturnTargetName                  string = "RETURN"

	AppendFlag             string = "-A"
	DestinationFlag        string = "-d"
	SourceFlag             string = "-s"
	JumpFlag               string = "-j"
	ProtocolFlag           string = "-p"
	MatchFlag              string = "-m"
	MatchParamComment      string = "comment"
	CommentFlag            string = "--comment"
	MatchParamMark         string = "mark"
	MarkFlag               string = "--mark"
	KuberboatMarkParam     string = "0x10000/0x10000"
	KuberboatHostMarkParam string = "0x8000/0x8000"
	MatchParamStatistic    string = "statistic"
	ModeFlag               string = "--mode"
	ModeParamNth           string = "nth"
	EveryFlag              string = "--every"
	PacketFlag             string = "--packet"
	PacketParamZero        string = "0"
	DNatDestinationFlag    string = "--to-destination"
	SNatDestinationFlag    string = "--to-source"
	DestinationPortFlag    string = "--dport"
	RandomFullyFlag        string = "--random-fully"
	SetXMarkFlag           string = "--set-xmark"
	KuberboatXMarkParam1   string = "0x10000/0"
	KuberboatXMarkParam2   string = "0x10000/0x10000"
	KuberboatXMarkParam3   string = "0x8000/0"
	KuberboatXMarkParam4   string = "0x8000/0x8000"
	MatchParamNot          string = "!"
)

// IPTablesClient provides APIs to manage kernel iptables for service.
type IPTablesClient interface {
	// InitServiceIPTables creates iptables chains for overall kuberboat service. It also adds an MASQUERADE
	// rule in the created chain.
	InitServiceIPTables() error
	// CreateServiceChain creates an iptables chain for one service port mapping.
	CreateServiceChain() string
	// ApplyServiceChain adds a rule to KUBERBOAT-SERVICES chain, jumping to a service chain.
	ApplyServiceChain(serviceName string, clusterIP string, serviceChainName string, port uint16) error
	// CreatePodChain creates an iptables chain for a pod in one service port mapping.
	CreatePodChain() string
	// ApplyPodChainRules adds jump-to-mark rules and a DNAT rule to a pod chain.
	ApplyPodChainRules(podChainName string, podIP string, targetPort uint16, sameHost bool) error
	// ApplyPodChain inserts a rule to KUBERBOAT-SVC-<serviceChainID> chain, jumping to a chain for pod.
	// num is the sequence number of this pod in the service, used for round robin.
	ApplyPodChain(serviceName string, serviceChainName string, podName string, podChainName string, num int) error
	// DeleteServiceChain clears and deletes an iptables chain for service and its relevant rules.
	DeleteServiceChain(serviceName string, clusterIP string, serviceChainName string, port uint16) error
	// DeletePodChain clears and deletes an iptables chain for service and its relevant rules.
	DeletePodChain(podName string, podChainName string) error
	// ClearServiceChain clears an iptables chain for service.
	ClearServiceChain(serviceName string, serviceChainName string) error
}

type iptablesClientInner struct {
	hostINetIP string
	flannelIP  string
	iptables   *iptables.IPTables
}

func NewIptablesClient(hostINetIP string, flannelIP string) (IPTablesClient, error) {
	iptables, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(hostINetIP); ip == nil {
		return nil, fmt.Errorf("invalid host inet ip %s", hostINetIP)
	}
	if ip := net.ParseIP(flannelIP); ip == nil {
		return nil, fmt.Errorf("invalid flannel ip %s", flannelIP)
	}
	return &iptablesClientInner{
		hostINetIP: hostINetIP,
		flannelIP:  flannelIP,
		iptables:   iptables,
	}, nil
}

func (ic *iptablesClientInner) InitServiceIPTables() error {
	// 1. KUBERBOAT-SERVICES

	// Create a chain named KUBERBOAT-SERVICES in nat table.
	ic.iptables.NewChain(NatTableName, KuberboatServicesChainName)

	// Find out whether the rule exists in PREROUTING chain.
	exist, err := ic.iptables.Exists(
		NatTableName,
		PreroutingChainName,
		JumpFlag,
		KuberboatServicesChainName,
	)
	if err != nil {
		return err
	}

	if !exist {
		// If the rule does not exist in PREROUTING chain, then insert it.
		err = ic.iptables.Insert(
			NatTableName,
			PreroutingChainName,
			1,
			JumpFlag,
			KuberboatServicesChainName,
		)
		if err != nil {
			return fmt.Errorf(
				"error when initializing %s chain: %v",
				KuberboatServicesChainName,
				err,
			)
		}
	}

	// Find out whether the rule exists in OUTPUT chain.
	exist, err = ic.iptables.Exists(
		NatTableName,
		OutputChainName,
		JumpFlag,
		KuberboatServicesChainName,
	)
	if err != nil {
		return err
	}

	if !exist {
		// If the rule does not exist in OUTPUT chain, then insert it.
		err = ic.iptables.Insert(
			NatTableName,
			OutputChainName,
			1,
			JumpFlag,
			KuberboatServicesChainName,
		)
		if err != nil {
			return fmt.Errorf(
				"error when initializing %s chain: %v",
				KuberboatServicesChainName,
				err,
			)
		}
	}

	// 2. KUBERBOAT-POSTROUTING

	// Create a chain named KUBERBOAT-POSTROUTING in nat table.
	ic.iptables.NewChain(NatTableName, KuberboatPostroutingChainName)

	// Find out whether the rule exists in POSTROUTING chain.
	exist, err = ic.iptables.Exists(
		NatTableName,
		PostroutingChainName,
		JumpFlag,
		KuberboatPostroutingChainName,
	)
	if err != nil {
		return err
	}

	if !exist {
		// If the rule does not exist in POSTROUTING chain, then insert it.
		err = ic.iptables.Insert(
			NatTableName,
			PostroutingChainName,
			1,
			JumpFlag,
			KuberboatPostroutingChainName,
		)
		if err != nil {
			return fmt.Errorf(
				"error when initializing %s chain: %v",
				KuberboatPostroutingChainName,
				err,
			)
		}

		// -A KUBERBOAT-POSTROUTING -m mark ! --mark 0x10000/0x10000 -j RETURN
		err = ic.iptables.AppendUnique(
			NatTableName,
			KuberboatPostroutingChainName,
			MatchFlag,
			MatchParamMark,
			MatchParamNot,
			MarkFlag,
			KuberboatMarkParam,
			JumpFlag,
			ReturnTargetName,
		)
		if err != nil {
			return fmt.Errorf(
				"error when applying RETURN rule in %s chain: %v",
				KuberboatPostroutingChainName,
				err,
			)
		}

		// -A KUBERBOAT-POSTROUTING -j MARK --set-xmark 0x10000/0
		err = ic.iptables.AppendUnique(
			NatTableName,
			KuberboatPostroutingChainName,
			JumpFlag,
			MarkTargetName,
			SetXMarkFlag,
			KuberboatXMarkParam1,
		)
		if err != nil {
			return fmt.Errorf(
				"error when applying MARK rule in %s chain: %v",
				KuberboatPostroutingChainName,
				err,
			)
		}

		// -A KUBERBOAT-POSTROUTING -j MASQUERADE --random-fully
		err = ic.iptables.AppendUnique(
			NatTableName,
			KuberboatPostroutingChainName,
			JumpFlag,
			MasqueradeTargetName,
			RandomFullyFlag,
		)
		if err != nil {
			return fmt.Errorf(
				"error when applying MASQUERADE rule in %s chain: %v",
				KuberboatPostroutingChainName,
				err,
			)
		}
	}

	// 3. KUBERBOAT-MARK-MASQ

	// Create a chain named KUBERBOAT-MARK-MASQ in nat table.
	ic.iptables.NewChain(NatTableName, KuberboatMarkChainName)

	// Add a MARK rule in KUBERBOAT-MARK-MASQ chain.
	err = ic.iptables.AppendUnique(
		NatTableName,
		KuberboatMarkChainName,
		JumpFlag,
		MarkTargetName,
		SetXMarkFlag,
		KuberboatXMarkParam2,
	)
	if err != nil {
		return fmt.Errorf(
			"error when applying MARK rule in %s chain: %v",
			KuberboatMarkChainName,
			err,
		)
	}

	// 4. KUBERBOAT-HOST-POSTROUTING

	// Create a chain named KUBERBOAT-HOST-POSTROUTING in nat table.
	ic.iptables.NewChain(NatTableName, KuberboatHostPostroutingChainName)

	// Find out whether the rule exists in POSTROUTING chain.
	exist, err = ic.iptables.Exists(
		NatTableName,
		PostroutingChainName,
		SourceFlag,
		ic.hostINetIP,
		JumpFlag,
		KuberboatHostPostroutingChainName,
	)
	if err != nil {
		return err
	}

	if !exist {
		// If the rule does not exist in POSTROUTING chain, then insert it.
		err = ic.iptables.Insert(
			NatTableName,
			PostroutingChainName,
			1,
			SourceFlag,
			ic.hostINetIP,
			JumpFlag,
			KuberboatHostPostroutingChainName,
		)
		if err != nil {
			return fmt.Errorf(
				"error when initializing %s chain: %v",
				KuberboatHostPostroutingChainName,
				err,
			)
		}

		// -A KUBERBOAT-HOST-POSTROUTING -m mark ! --mark 0x8000/0x8000 -j RETURN
		err = ic.iptables.AppendUnique(
			NatTableName,
			KuberboatHostPostroutingChainName,
			MatchFlag,
			MatchParamMark,
			MatchParamNot,
			MarkFlag,
			KuberboatHostMarkParam,
			JumpFlag,
			ReturnTargetName,
		)
		if err != nil {
			return fmt.Errorf(
				"error when applying RETURN rule in %s chain: %v",
				KuberboatHostPostroutingChainName,
				err,
			)
		}

		// -A KUBERBOAT-HOST-POSTROUTING -j MARK --set-xmark 0x8000/0
		err = ic.iptables.AppendUnique(
			NatTableName,
			KuberboatHostPostroutingChainName,
			JumpFlag,
			MarkTargetName,
			SetXMarkFlag,
			KuberboatXMarkParam3,
		)
		if err != nil {
			return fmt.Errorf(
				"error when applying MARK rule in %s chain: %v",
				KuberboatHostPostroutingChainName,
				err,
			)
		}

		// -A KUBERBOAT-HOST-POSTROUTING -j SNAT --to-source <Flannel IP>
		err = ic.iptables.AppendUnique(
			NatTableName,
			KuberboatHostPostroutingChainName,
			JumpFlag,
			SNatChainName,
			SNatDestinationFlag,
			ic.flannelIP,
		)
		if err != nil {
			return fmt.Errorf(
				"error when applying SNAT rule in %s chain: %v",
				KuberboatHostPostroutingChainName,
				err,
			)
		}
	}

	// 5. KUBERBOAT-HOST-MARK-MASQ

	// Create a chain named KUBERBOAT-HOST-MARK-MASQ in nat table.
	ic.iptables.NewChain(NatTableName, KuberboatHostMarkChainName)

	// Add a MARK rule in KUBERBOAT-HOST-MARK-MASQ chain.
	err = ic.iptables.AppendUnique(
		NatTableName,
		KuberboatHostMarkChainName,
		JumpFlag,
		MarkTargetName,
		SetXMarkFlag,
		KuberboatXMarkParam4,
	)
	if err != nil {
		return fmt.Errorf(
			"error when applying MARK rule in %s chain: %v",
			KuberboatMarkChainName,
			err,
		)
	}

	return nil
}

func (ic *iptablesClientInner) CreateServiceChain() string {
	// Create a chain named KUBERBOAT-SVC-<serviceChainID> in nat table.
	serviceChainID := strings.ToUpper(uuid.New().String()[:8])
	newChainName := KuberboatServiceChainPrefix + serviceChainID
	ic.iptables.NewChain(NatTableName, newChainName)
	return newChainName
}

func (ic *iptablesClientInner) ApplyServiceChain(
	serviceName string,
	clusterIP string,
	serviceChainName string,
	port uint16,
) error {
	if net.ParseIP(clusterIP) == nil {
		return fmt.Errorf("cluster IP %s is not valid", clusterIP)
	}

	// Add a rule of jumping to KUBERBOAT-SVC-<serviceChainID> chain.
	err := ic.iptables.Insert(
		NatTableName,
		KuberboatServicesChainName,
		1,
		ProtocolFlag,
		TCPProtocol,
		DestinationFlag,
		clusterIP,
		MatchFlag,
		TCPProtocol,
		DestinationPortFlag,
		fmt.Sprint(port),
		MatchFlag,
		MatchParamComment,
		CommentFlag,
		serviceName,
		JumpFlag,
		serviceChainName,
	)
	if err != nil {
		return fmt.Errorf(
			"error when applying iptables chain for service %s",
			serviceName,
		)
	}

	return nil
}

func (ic *iptablesClientInner) CreatePodChain() string {
	// Create a chain named KUBERBOAT-SEP-<podChainID> in nat table.
	podChainID := strings.ToUpper(uuid.New().String()[:8])
	newChainName := KuberboatPodChainPrefix + podChainID
	ic.iptables.NewChain(NatTableName, newChainName)
	return newChainName
}

func (ic *iptablesClientInner) ApplyPodChainRules(
	podChainName string,
	podIP string,
	targetPort uint16,
	sameHost bool,
) error {
	if net.ParseIP(podIP) == nil {
		return fmt.Errorf("pod IP %s is not valid", podIP)
	}

	// Add a rule that jumps to KUBERBOAT-MARK-MASQ when the source IP is exactly the pod IP.
	err := ic.iptables.AppendUnique(
		NatTableName,
		podChainName,
		SourceFlag,
		podIP,
		JumpFlag,
		KuberboatMarkChainName,
	)
	if err != nil {
		return fmt.Errorf(
			"error when applying jump-to-mask rule for pod IP %s: %v",
			podIP,
			err,
		)
	}

	// For chains of pods on the same host as Kubelet, we do nothing. For chains of other pods, we add
	// a mask chain for doing SNAT later.
	if !sameHost {
		err := ic.iptables.AppendUnique(
			NatTableName,
			podChainName,
			SourceFlag,
			ic.hostINetIP,
			JumpFlag,
			KuberboatHostMarkChainName,
		)
		if err != nil {
			return fmt.Errorf(
				"error when applying jump-to-mask rule for host IP %s: %v",
				ic.hostINetIP,
				err,
			)
		}
	}

	// Add a DNAT rule to the pod chain.
	destination := fmt.Sprintf("%s:%d", podIP, targetPort)
	err = ic.iptables.AppendUnique(
		NatTableName,
		podChainName,
		ProtocolFlag,
		TCPProtocol,
		JumpFlag,
		DNatChainName,
		DNatDestinationFlag,
		destination,
	)
	if err != nil {
		return fmt.Errorf(
			"error when applying iptables DNAT rule for pod IP %s: %v",
			podIP,
			err,
		)
	}

	return nil
}

func (ic *iptablesClientInner) ApplyPodChain(
	serviceName string,
	serviceChainName string,
	podName string,
	podChainName string,
	num int,
) error {
	// Add a rule of jumping to KUBERBOAT-SEP-<podChainID> chain.
	err := ic.iptables.Insert(
		NatTableName,
		serviceChainName,
		1,
		MatchFlag,
		MatchParamComment,
		CommentFlag,
		podName,
		MatchFlag,
		MatchParamStatistic,
		ModeFlag,
		ModeParamNth,
		EveryFlag,
		fmt.Sprint(num),
		PacketFlag,
		PacketParamZero,
		JumpFlag,
		podChainName,
	)
	if err != nil {
		return fmt.Errorf(
			"error when applying iptables chain for pod %s in service %s: %v",
			podName,
			serviceName,
			err,
		)
	}

	return nil
}

func (ic *iptablesClientInner) DeleteServiceChain(
	serviceName string,
	clusterIP string,
	serviceChainName string,
	port uint16,
) error {
	// Delete the rule that jumps to KUBERBOAT-SVC-<serviceChainID> chain in KUBERBOAT-SERVICES chain.
	err := ic.iptables.DeleteIfExists(
		NatTableName,
		KuberboatServicesChainName,
		ProtocolFlag,
		TCPProtocol,
		DestinationFlag,
		clusterIP,
		MatchFlag,
		TCPProtocol,
		DestinationPortFlag,
		fmt.Sprint(port),
		MatchFlag,
		MatchParamComment,
		CommentFlag,
		serviceName,
		JumpFlag,
		serviceChainName,
	)
	if err != nil {
		return fmt.Errorf(
			"error when deleting iptables rule for service %s: %v",
			serviceName,
			err,
		)
	}

	// Clear and delete KUBERBOAT-SVC-<serviceChainID> chain.
	err = ic.iptables.ClearAndDeleteChain(NatTableName, serviceChainName)
	if err != nil {
		return fmt.Errorf(
			"error when deleting iptables chain for service %s: %v",
			serviceName,
			err,
		)
	}

	return nil
}

func (ic *iptablesClientInner) DeletePodChain(podName string, podChainName string) error {
	// Clear and delete KUBERBOAT-SEP-<podChainID> chain.
	err := ic.iptables.ClearAndDeleteChain(NatTableName, podChainName)
	if err != nil {
		return fmt.Errorf(
			"error when deleting iptables chain for pod %s: %v",
			podName,
			err,
		)
	}
	return nil
}

func (ic *iptablesClientInner) ClearServiceChain(serviceName string, serviceChainName string) error {
	err := ic.iptables.ClearChain(NatTableName, serviceChainName)
	if err != nil {
		return fmt.Errorf(
			"error when clearing iptables chain for service %s: %v",
			serviceName,
			err,
		)
	}
	return nil
}
