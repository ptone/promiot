package promiot

import (
	dto "github.com/prometheus/client_model/go"
)

// definition of a simple array proto
type MetricBundle struct {
	// unix UTC time in nanoseconds
	// note that while nanosecond resolution is not needed or likely to be accurate
	// it is the easiest way to pass time around as single value in go
	// time.Now().UnixNano()
	BundleTimestamp int64               `protobuf:"varint,1,opt,name=bundle_timestamp" json:"bundle_timestamp,omitempty"`
	Families        []*dto.MetricFamily `protobuf:"bytes,2,rep,name=families" json:"families,omitempty"`
}

func (m *MetricBundle) Reset()         { *m = MetricBundle{} }
func (m *MetricBundle) String() string { return "" }
func (*MetricBundle) ProtoMessage()    {}
