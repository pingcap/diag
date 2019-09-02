package metric

import (
	influxdb "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
)

type batchWriter struct {
	cli       influxdb.Client
	threshold int
	batch     influxdb.BatchPoints
}

func NewBatchWriter(cli influxdb.Client, threshold int) *batchWriter {
	return &batchWriter{cli: cli, threshold: threshold}
}

func (bw *batchWriter) Write(db string, point *influxdb.Point) error {
	if bw.batch == nil {
		if batch, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  db,
			Precision: "s",
		}); err != nil {
			return err
		} else {
			bw.batch = batch
		}
	}

	bw.batch.AddPoint(point)
	if len(bw.batch.Points()) == bw.threshold {
		if err := bw.cli.Write(bw.batch); err != nil {
			return err
		} else {
			bw.batch = nil
		}
	}
	return nil
}

func (bw *batchWriter) Close() {
	if bw.batch == nil {
		return
	}

	if err := bw.cli.Write(bw.batch); err != nil {
		log.Error("write metric batch:", err)
	}
}
