package collector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type nodesInClusterCollector struct {
	logger log.Logger
	client *http.Client
	url    *url.URL
	node   string

	up                              prometheus.Gauge
	totalScrapes, jsonParseFailures prometheus.Counter

	nodesexists *prometheus.Desc
}

//You must create a constructor for you collector that
//initializes every descriptor and returns a pointer to the collector

func NewNodeInCluster(logger log.Logger, client *http.Client, url *url.URL, node string) *nodesInClusterCollector {
	subname := "node_in_cluster"
	return &nodesInClusterCollector{
		logger: logger,
		client: client,
		url:    url,
		node:   node,

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, subname, "up"),
			Help: "Was the last scrape of the Elasticsearch cluster node endpoint successful.",
		}),

		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, subname, "total_scrapes"),
			Help: "Current total ElasticSearch cluster node scrapes.",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, subname, "json_parse_failures"),
			Help: "Number of errors while parsing JSON.",
		}),

		//Static struc from func
		nodesexists: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subname, "nodesexists"),
			"Show if node exists in cluster. 1 it exists. 0 if it is not in the cluster.",
			[]string{"cluster", "name"}, nil,
		),
	}
}

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel.
func (c *nodesInClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up.Desc()
	ch <- c.totalScrapes.Desc()
	ch <- c.jsonParseFailures.Desc()
	ch <- c.nodesexists
}

func (c *nodesInClusterCollector) fetchAndDecodeClusterNodes() (clusterNodesResponse, error) {
	var chr clusterNodesResponse

	u := *c.url
	u.Path = path.Join(u.Path, "/_cluster/state/nodes")
	res, err := c.client.Get(u.String())
	if err != nil {
		return chr, fmt.Errorf("failed to get nodes from cluster from %s://%s:%s%s: %s",
			u.Scheme, u.Hostname(), u.Port(), u.Path, err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			_ = level.Warn(c.logger).Log(
				"msg", "failed to close http.Client",
				"err", err,
			)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return chr, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(&chr); err != nil {
		c.jsonParseFailures.Inc()
		return chr, err
	}

	return chr, nil
}

// Collect collects ClusterHealth metrics.
func (c *nodesInClusterCollector) Collect(ch chan<- prometheus.Metric) {
	var err error
	c.totalScrapes.Inc()
	defer func() {
		ch <- c.up
		ch <- c.totalScrapes
		ch <- c.jsonParseFailures
	}()

	clusterNodesResp, err := c.fetchAndDecodeClusterNodes()
	if err != nil {
		c.up.Set(0)
		_ = level.Warn(c.logger).Log(
			"msg", "failed to fetch and decode node in cluster status",
			"err", err,
		)
		return
	}
	c.up.Set(1)

	/* This is proof of concept. This shouldn't be like this.
	At least this variable shouldn't be declared like this and should've been passed as a pointer from NewNodeInCluster func. */
	var metricValue float64
	for _, node := range clusterNodesResp.Nodes {
		if strings.Contains(node.Name, c.node) {
			metricValue = 1
		}
	}

	//Static metric. If node is not in cluster metricValue is 0 and no value is passed into this
	ch <- prometheus.MustNewConstMetric(
		c.nodesexists,
		prometheus.CounterValue,
		metricValue,
		clusterNodesResp.ClusterName, c.node,
	)
}
