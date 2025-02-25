package kinesis

import (
	"crypto/md5"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/aws/aws-xray-sdk-go/xray"
	rec "github.com/awslabs/kinesis-aggregation/go/records"

	//nolint: staticcheck // not ready to switch package
	"github.com/golang/protobuf/proto"
)

const (
	kplMagicLen   = 4  // Length of magic header for KPL Aggregate Record checking.
	kplDigestSize = 16 // MD5 Message size for protobuf.
	kplMaxBytes   = 1024 * 1024
	kplMaxCount   = 10000
)

// Aggregate produces a KPL-compliant Kinesis record
type Aggregate struct {
	Record       *rec.AggregatedRecord
	Count        int
	MaxCount     int
	MaxSize      int
	PartitionKey string
}

// New creates a new Kinesis record with default values
// https://github.com/awslabs/kinesis-aggregation/blob/398fbd4b430d4bf590431b301d03cbbc94279cef/python/aws_kinesis_agg/aggregator.py#L167
func (a *Aggregate) New() {
	a.Record = &rec.AggregatedRecord{}
	a.Count = 0

	if a.MaxCount == 0 {
		a.MaxCount = kplMaxCount
	}
	if a.MaxCount > kplMaxCount {
		a.MaxCount = kplMaxCount
	}

	if a.MaxSize == 0 {
		a.MaxSize = kplMaxBytes
	}
	if a.MaxSize > kplMaxBytes {
		a.MaxSize = kplMaxBytes
	}

	a.PartitionKey = ""
	a.Record.PartitionKeyTable = make([]string, 0)
}

func varIntSize(i int) int {
	if i == 0 {
		return 1
	}

	var needed int
	for i > 0 {
		needed++
		i >>= 1
	}

	bytes := needed / 7
	if needed%7 > 0 {
		bytes++
	}

	return bytes
}

func (a *Aggregate) calculateRecordSize(data []byte, partitionKey string) int {
	var recordSize int
	// https://github.com/awslabs/kinesis-aggregation/blob/398fbd4b430d4bf590431b301d03cbbc94279cef/python/aws_kinesis_agg/aggregator.py#L344-L349
	pkSize := 1 + varIntSize(len(partitionKey)) + len(partitionKey)
	recordSize += pkSize
	// https://github.com/awslabs/kinesis-aggregation/blob/398fbd4b430d4bf590431b301d03cbbc94279cef/python/aws_kinesis_agg/aggregator.py#L362-L364
	pkiSize := 1 + varIntSize(a.Count)
	recordSize += pkiSize
	// https://github.com/awslabs/kinesis-aggregation/blob/398fbd4b430d4bf590431b301d03cbbc94279cef/python/aws_kinesis_agg/aggregator.py#L371-L374
	dataSize := 1 + varIntSize(len(data)) + len(data)
	recordSize += dataSize
	// https://github.com/awslabs/kinesis-aggregation/blob/398fbd4b430d4bf590431b301d03cbbc94279cef/python/aws_kinesis_agg/aggregator.py#L376-L378
	recordSize = recordSize + 1 + varIntSize(pkiSize+dataSize)

	// input record size + current aggregated record size + 4 byte magic header + 16 byte MD5 digest
	return recordSize + a.Record.XXX_Size() + 20
}

// Add inserts a Kinesis record into an aggregated Kinesis record
// https://github.com/awslabs/kinesis-aggregation/blob/398fbd4b430d4bf590431b301d03cbbc94279cef/python/aws_kinesis_agg/aggregator.py#L382
func (a *Aggregate) Add(data []byte, partitionKey string) bool {
	// https://docs.aws.amazon.com/streams/latest/dev/key-concepts.html#partition-key
	if len(partitionKey) > 256 {
		partitionKey = partitionKey[0:256]
	}

	// grab the first parition key in the set of events
	if a.PartitionKey == "" {
		a.PartitionKey = partitionKey
	}

	if a.Count > a.MaxCount {
		return false
	}

	newSize := a.calculateRecordSize(data, partitionKey)
	if newSize > a.MaxSize {
		return false
	}

	pki := uint64(a.Count)
	r := &rec.Record{
		PartitionKeyIndex: &pki,
		Data:              data,
	}

	a.Record.Records = append(a.Record.Records, r)
	a.Record.PartitionKeyTable = append(a.Record.PartitionKeyTable, partitionKey)
	a.Count++

	return true
}

// Get returns a KPL-compliant compressed Kinesis record
// https://github.com/awslabs/kinesis-aggregation/blob/398fbd4b430d4bf590431b301d03cbbc94279cef/python/aws_kinesis_agg/aggregator.py#L293
func (a *Aggregate) Get() []byte {
	data, _ := proto.Marshal(a.Record)
	md5Hash := md5.Sum(data)

	record := []byte("\xf3\x89\x9a\xc2")
	record = append(record, data...)
	record = append(record, md5Hash[:]...)

	return record
}

// ConvertEventsRecords converts Kinesis records between the Lambda and Go SDK packages. This is required for deaggregating Kinesis records processed by AWS Lambda.
func ConvertEventsRecords(records []events.KinesisEventRecord) []*kinesis.Record {
	output := make([]*kinesis.Record, 0)

	for _, r := range records {
		// ApproximateArrivalTimestamp is events.SecondsEpochTime which serializes time.Time
		ts := r.Kinesis.ApproximateArrivalTimestamp.UTC()
		output = append(output, &kinesis.Record{
			ApproximateArrivalTimestamp: &ts,
			Data:                        r.Kinesis.Data,
			EncryptionType:              &r.Kinesis.EncryptionType,
			PartitionKey:                &r.Kinesis.PartitionKey,
			SequenceNumber:              &r.Kinesis.SequenceNumber,
		})
	}

	return output
}

// New returns a configured Kinesis client.
func New() *kinesis.Kinesis {
	conf := aws.NewConfig()

	// provides forward compatibility for the Go SDK to support env var configuration settings
	// https://github.com/aws/aws-sdk-go/issues/4207
	max, found := os.LookupEnv("AWS_MAX_ATTEMPTS")
	if found {
		m, err := strconv.Atoi(max)
		if err != nil {
			panic(err)
		}

		conf = conf.WithMaxRetries(m)
	}

	c := kinesis.New(
		session.Must(session.NewSession()),
		conf,
	)

	if _, ok := os.LookupEnv("AWS_XRAY_DAEMON_ADDRESS"); ok {
		xray.AWS(c.Client)
	}

	return c
}

// API wraps the Kinesis API interface.
type API struct {
	Client kinesisiface.KinesisAPI
}

// Setup creates a new Kinesis client.
func (a *API) Setup() {
	a.Client = New()
}

// IsEnabled returns true if the client is enabled and ready for use.
func (a *API) IsEnabled() bool {
	return a.Client != nil
}

// PutRecord is a convenience wrapper for putting a record into a Kinesis stream.
func (a *API) PutRecord(ctx aws.Context, data []byte, stream, partitionKey string) (*kinesis.PutRecordOutput, error) {
	resp, err := a.Client.PutRecordWithContext(
		ctx,
		&kinesis.PutRecordInput{
			Data:         data,
			StreamName:   aws.String(stream),
			PartitionKey: aws.String(partitionKey),
		})
	if err != nil {
		return nil, fmt.Errorf("putrecord stream %s partitionkey %s: %v", stream, partitionKey, err)
	}

	return resp, nil
}

// ActiveShards returns the number of in-use shards for a Kinesis stream.
func (a *API) ActiveShards(ctx aws.Context, stream string) (int64, error) {
	var shards int64
	params := &kinesis.ListShardsInput{
		StreamName: aws.String(stream),
	}

LOOP:
	for {
		output, err := a.Client.ListShardsWithContext(ctx, params)
		if err != nil {
			return 0, fmt.Errorf("listshards stream %s: %v", stream, err)
		}

		for _, s := range output.Shards {
			if end := s.SequenceNumberRange.EndingSequenceNumber; end == nil {
				shards++
			}
		}

		if output.NextToken != nil {
			params = &kinesis.ListShardsInput{
				NextToken: output.NextToken,
			}
		} else {
			break LOOP
		}
	}

	return shards, nil
}

// UpdateShards uniformly updates a Kinesis stream's shard count and returns when the update is complete.
func (a *API) UpdateShards(ctx aws.Context, stream string, shards int64) error {
	params := &kinesis.UpdateShardCountInput{
		StreamName:       aws.String(stream),
		TargetShardCount: aws.Int64(shards),
		ScalingType:      aws.String("UNIFORM_SCALING"),
	}
	if _, err := a.Client.UpdateShardCountWithContext(ctx, params); err != nil {
		return fmt.Errorf("updateshards stream %s shards %d: %v", stream, shards, err)
	}

	for {
		resp, err := a.Client.DescribeStreamSummaryWithContext(ctx,
			&kinesis.DescribeStreamSummaryInput{
				StreamName: aws.String(stream),
			})
		if err != nil {
			return fmt.Errorf("describestream stream %s: %v", stream, err)
		}

		if status := resp.StreamDescriptionSummary.StreamStatus; status != aws.String("UPDATING") {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

// GetTags recursively retrieves all tags for a Kinesis stream.
func (a *API) GetTags(ctx aws.Context, stream string) ([]*kinesis.Tag, error) {
	var tags []*kinesis.Tag
	var lastTag string

	for {
		req := &kinesis.ListTagsForStreamInput{
			StreamName: aws.String(stream),
		}

		if lastTag != "" {
			req.ExclusiveStartTagKey = aws.String(lastTag)
		}

		resp, err := a.Client.ListTagsForStreamWithContext(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("listtags stream %s: %v", stream, err)
		}

		tags = append(tags, resp.Tags...)
		lastTag = *resp.Tags[len(resp.Tags)-1].Key

		// enables recursion
		if !*resp.HasMoreTags {
			break
		}
	}

	return tags, nil
}
