package valkey

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"

	valkey "github.com/valkey-io/valkey-go"
)

type singleNodeOptions struct {
	Socket     *socketOptions `json:"socket,omitempty"`
	Username   string         `json:"username,omitempty"`
	Password   string         `json:"password,omitempty"` //nolint:gosec
	ClientName string         `json:"clientName,omitempty"`
	Database        int            `json:"database,omitempty"`
	Resp3           bool           `json:"resp3,omitempty"`
	ClientSideCache bool           `json:"clientSideCache,omitempty"`
}

func (opts singleNodeOptions) toNodeInfo() (nodeInfo, error) {
	ni := nodeInfo{
		username:   opts.Username,
		password:   opts.Password,
		clientName: opts.ClientName,
		selectDB:   opts.Database,
	}

	addr, tlsCfg, err := setSocketOptions(opts.Socket)
	if err != nil {
		return nodeInfo{}, err
	}

	ni.addr = addr
	ni.tlsConfig = tlsCfg

	return ni, nil
}

type socketOptions struct {
	Host string      `json:"host,omitempty"`
	Port int         `json:"port,omitempty"`
	TLS  *tlsOptions `json:"tls,omitempty"`
}

type tlsOptions struct {
	// TODO: Handle binary data (ArrayBuffer) for all these as well.
	CA   []string `json:"ca,omitempty"`
	Cert string   `json:"cert,omitempty"`
	Key  string   `json:"key,omitempty"`
}

type commonClusterOptions struct {
	ReadOnly        bool `json:"readOnly,omitempty"`
	Resp3           bool `json:"resp3,omitempty"`
	ClientSideCache bool `json:"clientSideCache,omitempty"`
}

type clusterNodesMapOptions struct {
	commonClusterOptions
	Nodes []*singleNodeOptions `json:"nodes,omitempty"`
}

type clusterNodesStringOptions struct {
	commonClusterOptions
	Nodes []string `json:"nodes,omitempty"`
}

type sentinelOptions struct {
	singleNodeOptions
	MasterName       string `json:"masterName,omitempty"`
	SentinelUsername string `json:"sentinelUsername,omitempty"`
	SentinelPassword string `json:"sentinelPassword,omitempty"`
}

// nodeInfo is used to track consistency when building options from object nodes.
type nodeInfo struct {
	addr       string
	username   string
	password   string
	clientName string
	selectDB   int
	tlsConfig  *tls.Config
}

// newOptionsFromObject validates and instantiates an options struct from its
// map representation as exported from sobek.Runtime.
//
// The "shared" key is extracted from the map before JSON decoding so that
// DisallowUnknownFields does not reject it.
func newOptionsFromObject(obj map[string]any) (valkey.ClientOption, bool, error) {
	shared, _ := obj["shared"].(bool)
	delete(obj, "shared")

	var options any
	if cluster, ok := obj["cluster"].(map[string]any); ok {
		obj = cluster
		nodes, ok := cluster["nodes"].([]any)
		if !ok {
			return valkey.ClientOption{}, false, fmt.Errorf("cluster nodes property must be an array; got %T", cluster["nodes"])
		}
		if len(nodes) == 0 {
			return valkey.ClientOption{}, false, errors.New("cluster nodes property cannot be empty")
		}
		switch nodes[0].(type) {
		case map[string]any:
			options = &clusterNodesMapOptions{}
		case string:
			options = &clusterNodesStringOptions{}
		default:
			return valkey.ClientOption{}, false, fmt.Errorf("cluster nodes array must contain string or object elements; got %T", nodes[0])
		}
	} else if _, ok := obj["masterName"]; ok {
		options = &sentinelOptions{}
	} else {
		options = &singleNodeOptions{}
	}

	jsonStr, err := json.Marshal(obj)
	if err != nil {
		return valkey.ClientOption{}, false, fmt.Errorf("unable to serialize options to JSON %w", err)
	}

	// Instantiate a JSON decoder which will error on unknown
	// fields. As a result, if the input map contains an unknown
	// option, this function will produce an error.
	decoder := json.NewDecoder(bytes.NewReader(jsonStr))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&options)
	if err != nil {
		return valkey.ClientOption{}, false, err
	}

	copts, err := toClientOption(options)
	return copts, shared, err
}

// newOptionsFromString parses the expected URL into a valkey.ClientOption.
// Note: The shared client option is not supported with URL strings.
// Use the object form with `shared: true` to enable shared mode.
func newOptionsFromString(url string) (valkey.ClientOption, error) {
	opts, err := valkey.ParseURL(url)
	if err != nil {
		return valkey.ClientOption{}, err
	}

	opts.AlwaysRESP2 = true
	opts.DisableCache = true

	return opts, nil
}

func readOptions(options any) (valkey.ClientOption, bool, error) {
	var (
		opts   valkey.ClientOption
		shared bool
		err    error
	)
	switch val := options.(type) {
	case string:
		opts, err = newOptionsFromString(val)
	case map[string]any:
		opts, shared, err = newOptionsFromObject(val)
	default:
		return valkey.ClientOption{}, false, fmt.Errorf("invalid options type: %T; expected string or object", val)
	}

	if err != nil {
		return valkey.ClientOption{}, false, fmt.Errorf("invalid options; reason: %w", err)
	}

	return opts, shared, nil
}

func toClientOption(options any) (valkey.ClientOption, error) {
	copts := valkey.ClientOption{
		AlwaysRESP2:  true,
		DisableCache: true,
	}

	switch o := options.(type) {
	case *clusterNodesMapOptions:
		setClusterOptions(&copts, &o.commonClusterOptions)
		if o.Resp3 {
			copts.AlwaysRESP2 = false
		}
		if o.ClientSideCache {
			copts.DisableCache = false
			copts.AlwaysRESP2 = false
		}

		var infos []nodeInfo
		for _, n := range o.Nodes {
			ni, err := n.toNodeInfo()
			if err != nil {
				return valkey.ClientOption{}, err
			}
			infos = append(infos, ni)
		}
		if err := setConsistentOptions(&copts, infos); err != nil {
			return valkey.ClientOption{}, err
		}
	case *clusterNodesStringOptions:
		setClusterOptions(&copts, &o.commonClusterOptions)
		if o.Resp3 {
			copts.AlwaysRESP2 = false
		}
		if o.ClientSideCache {
			copts.DisableCache = false
			copts.AlwaysRESP2 = false
		}

		var infos []nodeInfo
		for _, n := range o.Nodes {
			parsed, err := valkey.ParseURL(n)
			if err != nil {
				return valkey.ClientOption{}, err
			}
			ni := nodeInfo{
				username:  parsed.Username,
				password:  parsed.Password,
				selectDB:  parsed.SelectDB,
				tlsConfig: parsed.TLSConfig,
			}
			if len(parsed.InitAddress) > 0 {
				ni.addr = parsed.InitAddress[0]
			}
			infos = append(infos, ni)
		}
		if err := setConsistentOptions(&copts, infos); err != nil {
			return valkey.ClientOption{}, err
		}
	case *sentinelOptions:
		copts.Sentinel = valkey.SentinelOption{
			MasterSet: o.MasterName,
			Username:  o.SentinelUsername,
			Password:  o.SentinelPassword,
		}
		if o.Resp3 {
			copts.AlwaysRESP2 = false
		}
		if o.ClientSideCache {
			copts.DisableCache = false
			copts.AlwaysRESP2 = false
		}

		ni, err := o.toNodeInfo()
		if err != nil {
			return valkey.ClientOption{}, err
		}
		if err := setConsistentOptions(&copts, []nodeInfo{ni}); err != nil {
			return valkey.ClientOption{}, err
		}
	case *singleNodeOptions:
		if o.Resp3 {
			copts.AlwaysRESP2 = false
		}
		if o.ClientSideCache {
			copts.DisableCache = false
			copts.AlwaysRESP2 = false
		}

		ni, err := o.toNodeInfo()
		if err != nil {
			return valkey.ClientOption{}, err
		}
		if err := setConsistentOptions(&copts, []nodeInfo{ni}); err != nil {
			return valkey.ClientOption{}, err
		}
	default:
		panic(fmt.Sprintf("unexpected options type %T", options))
	}

	return copts, nil
}

// setConsistentOptions collects addresses into InitAddress and validates that
// Username, Password, ClientName, and SelectDB are consistent across all nodes.
// TLS config is taken from the first node that provides it.
//
//nolint:gocognit,cyclop
func setConsistentOptions(copts *valkey.ClientOption, infos []nodeInfo) error {
	for _, ni := range infos {
		copts.InitAddress = append(copts.InitAddress, ni.addr)

		// Only set the TLS config once. Note that this assumes the same config is
		// used in other single-node options, since doing the consistency check we
		// use for the other options would be tedious.
		if copts.TLSConfig == nil && ni.tlsConfig != nil {
			copts.TLSConfig = ni.tlsConfig
		}

		if copts.SelectDB != 0 && ni.selectDB != 0 && copts.SelectDB != ni.selectDB {
			return fmt.Errorf("inconsistent db option: %d != %d", copts.SelectDB, ni.selectDB)
		}
		if ni.selectDB != 0 {
			copts.SelectDB = ni.selectDB
		}

		if copts.Username != "" && ni.username != "" && copts.Username != ni.username {
			return fmt.Errorf("inconsistent username option: %s != %s", copts.Username, ni.username)
		}
		if ni.username != "" {
			copts.Username = ni.username
		}

		if copts.Password != "" && ni.password != "" && copts.Password != ni.password {
			return fmt.Errorf("inconsistent password option")
		}
		if ni.password != "" {
			copts.Password = ni.password
		}

		if copts.ClientName != "" && ni.clientName != "" && copts.ClientName != ni.clientName {
			return fmt.Errorf("inconsistent clientName option: %s != %s", copts.ClientName, ni.clientName)
		}
		if ni.clientName != "" {
			copts.ClientName = ni.clientName
		}
	}

	return nil
}

func setClusterOptions(copts *valkey.ClientOption, opts *commonClusterOptions) {
	if opts.ReadOnly {
		copts.SendToReplicas = func(cmd valkey.Completed) bool {
			return cmd.IsReadOnly()
		}
	}
}

// setSocketOptions extracts the address string and optional *tls.Config from
// socket options.
func setSocketOptions(sopts *socketOptions) (string, *tls.Config, error) {
	if sopts == nil {
		return "", nil, fmt.Errorf("empty socket options")
	}

	addr := fmt.Sprintf("%s:%d", sopts.Host, sopts.Port)

	var tlsCfg *tls.Config
	if sopts.TLS != nil {
		//#nosec G402
		tlsCfg = &tls.Config{}
		if len(sopts.TLS.CA) > 0 {
			caCertPool := x509.NewCertPool()
			for _, cert := range sopts.TLS.CA {
				caCertPool.AppendCertsFromPEM([]byte(cert))
			}
			tlsCfg.RootCAs = caCertPool
		}

		if sopts.TLS.Cert != "" && sopts.TLS.Key != "" {
			clientCertPair, err := tls.X509KeyPair([]byte(sopts.TLS.Cert), []byte(sopts.TLS.Key))
			if err != nil {
				return "", nil, err
			}
			tlsCfg.Certificates = []tls.Certificate{clientCertPair}
		}
	}

	return addr, tlsCfg, nil
}
