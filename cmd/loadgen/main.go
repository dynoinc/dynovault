package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/dynoinc/dynovault/feastle"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/valyala/histogram"
)

var (
	endpointURL       string
	bulkfillDuration  time.Duration
	steadyDuration    time.Duration
	numTables         int
	numParallelReader int
	numParallelWriter int
)

func main() {
	flag.StringVar(&endpointURL, "endpoint", "http://127.0.0.1:8779", "DynamoDB endpoint")
	flag.DurationVar(&bulkfillDuration, "bulkfill_duration", 1*time.Second, "Duration to run bulkfill for")
	flag.DurationVar(&steadyDuration, "steady_duration", 5*time.Second, "Duration to run steady load for")
	flag.IntVar(&numTables, "num_tables", 10, "Number of tables to create")
	flag.IntVar(&numParallelReader, "num_parallel_reader", 10, "Number of parallel readers")
	flag.IntVar(&numParallelWriter, "num_parallel_writer", 10, "Number of parallel writers")
	flag.Parse()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("ID", "SECRET_KEY", "TOKEN"),
		),
	)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %s", err))
	}

	db := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpointURL)
	})

	lg := newLoadgen(
		loadgenParams{
			numTables:         numTables,
			numParallelReader: numParallelReader,
			numParallelWriter: numParallelWriter,
		},
		db,
	)

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
	}()

	err = lg.BulkUpload(ctx, bulkfillDuration)
	if err != nil {
		panic(fmt.Errorf("failed to bulk upload: %s", err))
	}

	_ = lg.Run(ctx, steadyDuration)
}

type loadgenParams struct {
	numTables         int
	numParallelReader int
	numParallelWriter int
}

type loadgen struct {
	param  loadgenParams
	db     *dynamodb.Client
	tables []string
}

func newLoadgen(param loadgenParams, db *dynamodb.Client) *loadgen {
	return &loadgen{
		param: param,
		db:    db,
	}
}

func (l *loadgen) createTables(ctx context.Context, tables []string) error {
	beginTime := time.Now()
	log.Println("Creating tables.... len(tables):", len(tables))
	for _, tableName := range tables {
		_, err := l.db.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName: aws.String(tableName),
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("entity_id"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("entity_id"),
					KeyType:       types.KeyTypeHash,
				},
			},
		})
		if err != nil {
			log.Println("Failed to create table", tableName, err)
			return err
		}
	}

	for _, tableName := range tables {
		table, err := l.db.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return err
		}

		if *table.Table.TableName != tableName {
			panic(fmt.Errorf("table name mismatch... %s %s", *table.Table.TableName, tableName))
		}
	}

	log.Println("Created tables in...", time.Since(beginTime))
	return nil
}

func (l *loadgen) BulkUpload(ctx context.Context, duration time.Duration) error {
	l.tables = generateTableNames(l.param.numTables)
	err := l.createTables(ctx, l.tables)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(l.param.numParallelWriter)
	var writerErr error
	var numWrites int
	for i := 0; i < l.param.numParallelWriter; i++ {
		go func() {
			defer wg.Done()

			timer := time.NewTimer(duration)
			defer timer.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					return
				default:
				}
				err := l.doBatchWriteItem(ctx)
				if err != nil {
					writerErr = err
				}

				numWrites += 1
			}
		}()
	}
	wg.Wait()

	log.Printf(
		"Bulk uploaded %d items in %.02fs at %.0f/s\n",
		numWrites,
		duration.Seconds(),
		float64(numWrites)/duration.Seconds(),
	)
	return writerErr
}

// Run blocks until the context is cancelled.
func (l *loadgen) Run(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	r, w := 0, 0
	rp, wp := histogram.NewFast(), histogram.NewFast()

L:
	for {
		select {
		case <-timer.C:
			break L
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 1% write, 99% get
		switch rand.Intn(100) {
		case 0:
			t := time.Now()
			if err := l.doBatchWriteItem(ctx); err != nil {
				fmt.Printf("failed to write: %v", err)
			}
			w += 1
			wp.Update(float64(time.Since(t).Milliseconds()))
		default:
			t := time.Now()
			if err := l.doBatchGetItem(ctx); err != nil {
				fmt.Printf("failed to get: %v", err)
			}
			r += 1
			rp.Update(float64(time.Since(t).Milliseconds()))
		}
	}

	log.Printf("Ran steady load for %.02fs", steadyDuration.Seconds())
	log.Printf(
		"Reads: total %d, QPS %.02f/s, avg %.02fms, p99 %.02fms",
		r,
		float64(r)/steadyDuration.Seconds(),
		rp.Quantile(.5),
		rp.Quantile(.99),
	)
	log.Printf(
		"Writes: total %d, QPS %.02f/s, avg %.02fms, p99 %.02fms",
		w,
		float64(w)/steadyDuration.Seconds(),
		wp.Quantile(.5),
		wp.Quantile(.99),
	)
	return nil
}

func (l *loadgen) doBatchGetItem(ctx context.Context) error {
	numFeatures := rand.Int()%100 + 1
	features := make([]*feastle.FeastFeature, numFeatures)
	for i := 0; i < numFeatures; i++ {
		features[i] = feastle.GenerateRandomFeature(l.tables)
	}
	batchGetItemInput := feastle.NewBatchGetItemInput(features)
	_, err := l.db.BatchGetItem(ctx, batchGetItemInput)
	if err != nil {
		return err
	}
	return nil
}

func (l *loadgen) doBatchWriteItem(ctx context.Context) error {
	numFeatures := rand.Int()%100 + 1
	features := make([]*feastle.FeastFeature, numFeatures)
	for i := 0; i < numFeatures; i++ {
		features[i] = feastle.GenerateRandomFeature(l.tables)
	}
	batchWriteItemInput := feastle.NewBatchWriteItemInput(features)
	_, err := l.db.BatchWriteItem(ctx, batchWriteItemInput)
	if err != nil {
		return err
	}
	return nil
}

func generateTableNames(numTables int) []string {
	tableNames := make([]string, 0, numTables)
	for i := 0; i < numTables; i++ {
		tableNames = append(tableNames, fmt.Sprintf("feastle.driver_hourly_stats.%d", i))
	}
	return tableNames
}
