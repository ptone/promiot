package promiot

import (
	"fmt"
	"net/http"

	"github.com/prometheus/common/expfmt"

	dto "github.com/prometheus/client_model/go"
)

// FetchMetricFamilies retrieves metrics from the provided URL, decodes them
// into MetricFamily proto messages, and sends them to the provided channel. It
// returns after all MetricFamilies have been sent.
func FetchMetricFamilies(
	url string,
) (mfs []*dto.MetricFamily, err error) {
	transport := &http.Transport{}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GET request for URL %q failed: %v", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing GET request for URL %q failed: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET request for URL %q returned HTTP status %s", url, resp.Status)
	}
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("reading text format failed: %v", err)
	}
	// mfs = make([]*dto.MetricFamily)
	for _, f := range metricFamilies {
		mfs = append(mfs, f)
	}
	return mfs, nil
}
